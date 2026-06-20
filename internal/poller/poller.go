package poller

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/domainutil"
)

// trackProfileName marks hits stored by domain-tracker mode so they are
// attributable and distinguishable from keyword-profile hits.
const trackProfileName = "domain-track"

// PollStats tracks per-log polling progress, emitted once per batch.
type PollStats struct {
	CertsScanned int64

	// HitsFound counts domains that scored above zero and were stored.
	HitsFound int64

	// CurrentIndex is the next entry to fetch.
	CurrentIndex int64

	// TreeSize is the most recently observed tree size.
	TreeSize int64

	LogName string
}

// Poller continuously polls a single CT log, scoring domains and storing hits.
type Poller struct {
	client       *CTLogClient
	logName      string
	scorer       domain.Scorer
	store        domain.Store
	profile      *domain.Profile
	batchSize    int
	pollInterval time.Duration
	hitChan      chan<- domain.Hit
	statsChan    chan<- PollStats
	discardChan  chan<- string
	backtrack    int64
	minScore     int
	// session tags every stored hit for later --session querying; empty means untagged.
	session string
	// trackDomains holds tracker-mode apex targets, read-only so it shares lock-free.
	trackDomains []string
}

// NewPoller creates a poller for one CT log. backtrack > 0 starts at
// (tree_size - backtrack); discardChan receives zero-scored domains and may be nil.
func NewPoller(
	logURL string,
	logName string,
	scorer domain.Scorer,
	store domain.Store,
	profile *domain.Profile,
	batchSize int,
	pollInterval time.Duration,
	hitChan chan<- domain.Hit,
	statsChan chan<- PollStats,
	discardChan chan<- string,
	backtrack int64,
	minScore int,
	session string,
	trackDomains []string,
) *Poller {
	return &Poller{
		client:       NewCTLogClient(logURL),
		logName:      logName,
		scorer:       scorer,
		store:        store,
		profile:      profile,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		hitChan:      hitChan,
		statsChan:    statsChan,
		discardChan:  discardChan,
		backtrack:    backtrack,
		minScore:     minScore,
		session:      session,
		trackDomains: trackDomains,
	}
}

// Run polls for new entries, scoring and storing hits until the context is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	slog.Info("starting poller", "log", p.logName)

	sth, err := p.client.GetSTH(ctx)
	if err != nil {
		return fmt.Errorf("getting initial STH for %s: %w", p.logName, err)
	}

	currentIndex := p.startIndex(sth.TreeSize)
	slog.Info("poller initialized",
		"log", p.logName,
		"tree_size", sth.TreeSize,
		"backtrack", p.backtrack,
		"starting_at", currentIndex)

	stats := PollStats{
		LogName:      p.logName,
		CurrentIndex: currentIndex,
		TreeSize:     sth.TreeSize,
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("poller shutting down", "log", p.logName)
			return nil
		default:
		}

		next, stop := p.pollOnce(ctx, currentIndex, &stats)
		if stop {
			return nil
		}
		currentIndex = next
	}
}

// startIndex computes the entry index to begin polling from, applying the
// configured backtrack and clamping to zero.
func (p *Poller) startIndex(treeSize int64) int64 {
	if p.backtrack <= 0 {
		return treeSize
	}
	start := treeSize - p.backtrack
	if start < 0 {
		return 0
	}
	return start
}

// pollOnce refreshes the tree head, processes the next batch, and publishes
// stats. It returns the next index and whether the loop should stop (cancelled).
func (p *Poller) pollOnce(ctx context.Context, currentIndex int64, stats *PollStats) (next int64, stop bool) {
	sth, err := p.client.GetSTH(ctx)
	if err != nil {
		slog.Warn("failed to get STH, will retry", "log", p.logName, "error", err)
		return currentIndex, !p.sleep(ctx)
	}
	stats.TreeSize = sth.TreeSize

	if currentIndex >= sth.TreeSize {
		return currentIndex, !p.sleep(ctx)
	}

	end := currentIndex + int64(p.batchSize) - 1
	if end >= sth.TreeSize {
		end = sth.TreeSize - 1
	}

	entries, err := p.client.GetEntries(ctx, currentIndex, end)
	if err != nil {
		slog.Warn("failed to get entries, will retry",
			"log", p.logName, "start", currentIndex, "end", end, "error", err)
		return currentIndex, !p.sleep(ctx)
	}

	for _, entry := range entries {
		p.processEntry(ctx, entry, stats)
	}

	next = end + 1
	stats.CurrentIndex = next

	// Non-blocking send: drop the stats update if nobody is listening.
	select {
	case p.statsChan <- *stats:
	default:
	}
	return next, false
}

// processEntry parses a single CT log entry, extracts domains, scores them,
// and stores any hits.
func (p *Poller) processEntry(ctx context.Context, entry domain.CTLogEntry, stats *PollStats) {
	stats.CertsScanned++

	domains, cert, err := ParseCertDomains(entry)
	if err != nil {
		logParseWarning(entry.LogURL, entry.Index, err)
		return
	}

	for _, d := range domains {
		if p.processDomain(ctx, d, domains, cert) {
			stats.HitsFound++
		}
	}
}

// processDomain routes a domain through tracker or scoring mode and reports
// whether a hit was persisted.
func (p *Poller) processDomain(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	if len(p.trackDomains) > 0 {
		return p.processTracked(ctx, d, sanDomains, cert)
	}
	return p.processScored(ctx, d, sanDomains, cert)
}

// processTracked stores any tracked-target match unconditionally, bypassing
// minScore, the score==0 short-circuit, and skip-suffix filtering.
func (p *Poller) processTracked(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	if !p.matchesTrackedDomain(d) {
		return false
	}

	scored := p.scorer.ScoreWithCert(d, p.profile, certMeta(cert))
	hit := buildTrackedHit(d, sanDomains, cert, p.logName, p.session, scored)

	select {
	case p.hitChan <- hit:
	default:
	}

	if err := p.store.UpsertHit(ctx, hit); err != nil {
		slog.Warn("failed to upsert tracked hit", "domain", d, "error", err)
		return false
	}
	return true
}

// processScored scores a domain against the profile and persists it only when
// it meets the minimum score threshold.
func (p *Poller) processScored(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	scored := p.scorer.ScoreWithCert(d, p.profile, certMeta(cert))
	if scored.Score == 0 {
		p.publishDiscard(d)
		return false
	}

	hit := domain.Hit{
		Domain:        d,
		Score:         scored.Score,
		Severity:      scored.Severity,
		Keywords:      scored.MatchedKeywords,
		Signals:       scored.Signals,
		Category:      scored.Category,
		CTLog:         p.logName,
		Profile:       p.profile.Name,
		Session:       p.session,
		SANDomains:    sanDomains,
		CertNotBefore: cert.NotBefore,
		CreatedAt:     time.Now(),
	}
	if len(cert.Issuer.Organization) > 0 {
		hit.Issuer = cert.Issuer.Organization[0]
	}
	hit.IssuerCN = cert.Issuer.CommonName

	select {
	case p.hitChan <- hit:
	default:
	}

	// Unset minScore defaults the persist threshold to 4; --min-score raises it.
	threshold := p.minScore
	if threshold == 0 {
		threshold = 4
	}
	if scored.Score < threshold {
		return false
	}

	if err := p.store.UpsertHit(ctx, hit); err != nil {
		slog.Warn("failed to upsert hit", "domain", d, "error", err)
		return false
	}
	return true
}

// matchesTrackedDomain reports whether d matches any configured tracking target
// (apex + subdomains). The target slice is read-only, so this is lock-free.
func (p *Poller) matchesTrackedDomain(d string) bool {
	return domainutil.MatchesAnyTrackTarget(d, p.trackDomains)
}

// buildTrackedHit assembles a tracker-mode Hit, tagged with the trackProfileName
// marker so stored rows are attributable to tracker mode.
func buildTrackedHit(d string, sanDomains []string, cert *x509.Certificate, logName, session string, scored domain.ScoredDomain) domain.Hit {
	hit := domain.Hit{
		Domain:        d,
		Score:         scored.Score,
		Severity:      scored.Severity,
		Keywords:      scored.MatchedKeywords,
		Signals:       scored.Signals,
		Category:      "tracker",
		CTLog:         logName,
		Profile:       trackProfileName,
		Session:       session,
		SANDomains:    sanDomains,
		CertNotBefore: cert.NotBefore,
		CreatedAt:     time.Now(),
	}
	if len(cert.Issuer.Organization) > 0 {
		hit.Issuer = cert.Issuer.Organization[0]
	}
	hit.IssuerCN = cert.Issuer.CommonName
	return hit
}

// certMeta extracts cert metadata for the scoring heuristics; a nil cert yields
// a zero CertMeta, disabling those heuristics.
func certMeta(cert *x509.Certificate) domain.CertMeta {
	if cert == nil {
		return domain.CertMeta{}
	}
	meta := domain.CertMeta{SANCount: len(cert.DNSNames)}
	if !cert.NotAfter.IsZero() && !cert.NotBefore.IsZero() && cert.NotAfter.After(cert.NotBefore) {
		meta.ValidityDays = int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24)
	}
	// Join CN and org so free-CA substring matching sees both.
	issuerParts := make([]string, 0, 2)
	if cert.Issuer.CommonName != "" {
		issuerParts = append(issuerParts, cert.Issuer.CommonName)
	}
	if len(cert.Issuer.Organization) > 0 {
		issuerParts = append(issuerParts, cert.Issuer.Organization[0])
	}
	meta.Issuer = strings.Join(issuerParts, " ")
	return meta
}

// publishDiscard reports a zero-scored domain to the discard feed without
// blocking when no consumer is listening.
func (p *Poller) publishDiscard(d string) {
	if p.discardChan == nil {
		return
	}
	select {
	case p.discardChan <- d:
	default:
	}
}

// sleep waits one poll interval, returning false if the context was cancelled.
func (p *Poller) sleep(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(p.pollInterval):
		return true
	}
}
