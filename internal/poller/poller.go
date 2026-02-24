package poller

import (
	"context"
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
}

// NewPoller creates a poller for a single CT log endpoint.
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

	currentIndex := sth.TreeSize
	slog.Info("poller initialized",
		"log", p.logName,
		"tree_size", sth.TreeSize,
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

		// Refresh tree head.
		sth, err = p.client.GetSTH(ctx)
		if err != nil {
			slog.Warn("failed to get STH, will retry",
				"log", p.logName, "error", err)
			if err := p.sleep(ctx); err != nil {
				return nil
			}
			continue
		}
		stats.TreeSize = sth.TreeSize

		// No new entries.
		if currentIndex >= sth.TreeSize {
			if err := p.sleep(ctx); err != nil {
				return nil
			}
			continue
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
			if err := p.sleep(ctx); err != nil {
				return nil
			}
			continue
		}

		for _, entry := range entries {
			p.processEntry(ctx, entry, &stats)
		}

		currentIndex = end + 1
		stats.CurrentIndex = currentIndex

		// Send stats update.
		select {
		case p.statsChan <- stats:
		default:
			// Don't block if nobody is listening.
		}
	}
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
		scored := p.scorer.Score(d, p.profile)
		if scored.Score == 0 {
			continue
		}

		hit := domain.Hit{
			Domain:        d,
			Score:         scored.Score,
			Severity:      scored.Severity,
			Keywords:      scored.MatchedKeywords,
			CTLog:         p.logName,
			Profile:       p.profile.Name,
			SANDomains:    domains,
			CertNotBefore: cert.NotBefore,
		}

		// Populate issuer fields from certificate.
		if len(cert.Issuer.Organization) > 0 {
			hit.Issuer = cert.Issuer.Organization[0]
		}
		hit.IssuerCN = cert.Issuer.CommonName

		if err := p.store.UpsertHit(ctx, hit); err != nil {
			slog.Warn("failed to upsert hit",
				"domain", d, "error", err)
			continue
		}

		stats.HitsFound++

		// Send hit to TUI channel.
		select {
		case p.hitChan <- hit:
		default:
			// Don't block if channel is full.
		}
	}
}

// sleep waits for the poll interval or until the context is cancelled.
func (p *Poller) sleep(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.pollInterval):
		return nil
	}
}
