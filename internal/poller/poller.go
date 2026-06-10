package poller

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/domainutil"
)

// trackProfileName marks hits stored by domain-tracker mode so they are
// attributable and distinguishable from keyword-profile hits.
const trackProfileName = "domain-track"

// PollStats tracks per-log polling progress and throughput.
// One PollStats value is emitted per CT log after each batch of entries is processed.
type PollStats struct {
	// CertsScanned is the total number of certificate entries processed by this poller
	// since it started.
	CertsScanned int64

	// HitsFound is the number of domains that scored above zero and were stored.
	HitsFound int64

	// CurrentIndex is the current position in the CT log tree (next entry to fetch).
	CurrentIndex int64

	// TreeSize is the most recently observed tree size from the CT log get-sth endpoint.
	TreeSize int64

	// LogName is the human-readable name of the CT log this poller monitors.
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
	// session tags every stored hit so a run can be grouped and queried later
	// via --session. Empty means hits are stored without a session tag.
	session string
	// trackDomains holds normalized apex targets for tracker mode. When
	// non-empty, the poller stores every certificate whose domain matches a
	// target (apex + subdomains), unconditionally, and fast-paths non-matches.
	// Read-only after construction, so it is safe to share across the per-log
	// poller goroutines without locking.
	trackDomains []string
}

// NewPoller creates a poller for a single CT log endpoint. The backtrack
// parameter controls how many entries behind the current log tip to start.
// When backtrack > 0, the poller begins at (tree_size - backtrack), giving
// immediate results on launch. When backtrack == 0, the poller starts at
// the tip and waits for new entries. The discardChan receives domain names
// that scored zero and were not stored; it may be nil to skip discards.
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

// Run starts the polling loop. It fetches the current tree head, then
// continuously polls for new entries, scoring and storing hits. The loop
// exits when the context is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	slog.Info("starting poller", "log", p.logName)

	// Get initial tree head to determine starting position.
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

// pollOnce performs one polling iteration: it refreshes the tree head, fetches
// and processes the next batch of entries, and publishes a stats update. It
// returns the next index to poll and whether the loop should stop (the context
// was cancelled).
func (p *Poller) pollOnce(ctx context.Context, currentIndex int64, stats *PollStats) (next int64, stop bool) {
	// Refresh tree head.
	sth, err := p.client.GetSTH(ctx)
	if err != nil {
		slog.Warn("failed to get STH, will retry", "log", p.logName, "error", err)
		return currentIndex, !p.sleep(ctx)
	}
	stats.TreeSize = sth.TreeSize

	// No new entries.
	if currentIndex >= sth.TreeSize {
		return currentIndex, !p.sleep(ctx)
	}

	// Fetch entries in batches.
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

	// Send stats update without blocking if nobody is listening.
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

// processDomain routes a single domain through the active processing mode.
// In tracker mode (trackDomains non-empty) it stores every matching domain
// unconditionally; otherwise it applies the keyword-scoring path. It returns
// true when a hit was persisted (so the caller can increment its stats counter).
func (p *Poller) processDomain(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	if len(p.trackDomains) > 0 {
		return p.processTracked(ctx, d, sanDomains, cert)
	}
	return p.processScored(ctx, d, sanDomains, cert)
}

// processTracked handles tracker mode: non-matching domains take a fast path
// (no scoring, no feed, no store); a domain matching any tracked target is
// scored for informational fields, streamed to the feed, and stored
// unconditionally — bypassing minScore, the score==0 short-circuit, and
// skip-suffix filtering.
func (p *Poller) processTracked(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	if !p.matchesTrackedDomain(d) {
		return false
	}

	scored := p.scorer.ScoreWithCert(d, p.profile, certMeta(cert))
	hit := buildTrackedHit(d, sanDomains, cert, p.logName, p.session, scored)

	// Surface the tracked hit on the live feed for visibility.
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

// processScored is the keyword-scoring path: domains are scored against the
// profile, streamed to the feed, and persisted only when they meet the minimum
// score threshold.
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

	// Send all scored hits to the live feed for visibility.
	select {
	case p.hitChan <- hit:
	default:
	}

	// Only persist hits meeting the minimum score threshold. Default
	// (minScore=0) stores all scored hits; --min-score raises the bar.
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

// buildTrackedHit assembles a Hit for a tracker-mode match. Scoring fields are
// informational only; the row is tagged with the "domain-track" profile marker
// so stored rows are attributable to tracker mode.
func buildTrackedHit(d string, sanDomains []string, cert *x509.Certificate, logName, session string, scored domain.ScoredDomain) domain.Hit {
	hit := domain.Hit{
		Domain:        d,
		Score:         scored.Score,
		Severity:      scored.Severity,
		Keywords:      scored.MatchedKeywords,
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

// certMeta extracts the minimal certificate metadata used by the scoring
// engine's certificate-level heuristics: the SAN count and the validity period
// in whole days. A nil cert yields a zero CertMeta, which disables those
// heuristics.
func certMeta(cert *x509.Certificate) domain.CertMeta {
	if cert == nil {
		return domain.CertMeta{}
	}
	meta := domain.CertMeta{SANCount: len(cert.DNSNames)}
	if !cert.NotAfter.IsZero() && !cert.NotBefore.IsZero() && cert.NotAfter.After(cert.NotBefore) {
		meta.ValidityDays = int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24)
	}
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

// sleep waits for the poll interval or until the context is cancelled.
// It returns false if the context was cancelled (the poller should stop),
// and true if the full interval elapsed.
func (p *Poller) sleep(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(p.pollInterval):
		return true
	}
}
