package poller

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// Manager coordinates multiple Poller goroutines, one per configured CT log.
type Manager struct {
	cfg       *config.Config
	scorer    domain.Scorer
	store     domain.Store
	profile   *domain.Profile
	backtrack int64
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewManager creates a poller manager that will launch one poller per CT log
// from the config. The backtrack parameter controls how many entries behind
// the current log tip each poller starts at.
func NewManager(cfg *config.Config, scorer domain.Scorer, store domain.Store, profile *domain.Profile, backtrack int64) *Manager {
	return &Manager{
		cfg:       cfg,
		scorer:    scorer,
		store:     store,
		profile:   profile,
		backtrack: backtrack,
	}
}

// Start launches a polling goroutine for each CT log in the config. All
// pollers share the provided hit, stats, and discard channels. The
// discardChan receives domain names that scored zero; it may be nil to
// skip discard reporting. Returns immediately; pollers run until the
// context is cancelled.
func (m *Manager) Start(ctx context.Context, hitChan chan<- domain.Hit, statsChan chan<- PollStats, discardChan chan<- string) error {
	ctx, m.cancel = context.WithCancel(ctx)

	for _, logCfg := range m.cfg.CTLogs {
		p := NewPoller(
			logCfg.URL,
			logCfg.Name,
			m.scorer,
			m.store,
			m.profile,
			m.cfg.BatchSize,
			m.cfg.PollInterval,
			hitChan,
			statsChan,
			discardChan,
			m.backtrack,
		)

		m.wg.Add(1)
		go func(poller *Poller, name string) {
			defer m.wg.Done()
			if err := poller.Run(ctx); err != nil {
				slog.Error("poller exited with error", "log", name, "error", err)
			}
		}(p, logCfg.Name)
	}

	slog.Info("started pollers", "count", len(m.cfg.CTLogs))
	return nil
}

// Stop cancels all pollers and waits for them to exit.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	slog.Info("all pollers stopped")
}
