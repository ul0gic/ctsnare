package poller

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
)

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

// processDomain scores a single domain, streams scored hits to the live feed,
// and persists those meeting the minimum-score threshold. It returns true when
// a hit was persisted (so the caller can increment its stats counter).
func (p *Poller) processDomain(ctx context.Context, d string, sanDomains []string, cert *x509.Certificate) bool {
	scored := p.scorer.Score(d, p.profile)
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
