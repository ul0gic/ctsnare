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
	minScore  int
	// session tags every stored hit so a run can be grouped and queried later
	// via --session. Empty means hits are stored without a session tag.
	session string
	// trackDomains holds normalized apex targets for tracker mode, passed to
	// every poller. Empty means keyword-scoring mode.
	trackDomains []string
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewManager creates a manager that launches one poller per configured CT log.
// Non-empty trackDomains switches every poller to tracker mode; see NewPoller.
func NewManager(cfg *config.Config, scorer domain.Scorer, store domain.Store, profile *domain.Profile, backtrack int64, minScore int, session string, trackDomains []string) *Manager {
	return &Manager{
		cfg:          cfg,
		scorer:       scorer,
		store:        store,
		profile:      profile,
		backtrack:    backtrack,
		minScore:     minScore,
		session:      session,
		trackDomains: trackDomains,
	}
}

// Start launches one polling goroutine per CT log, returning immediately;
// pollers run until ctx is cancelled. discardChan may be nil to skip discards.
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
			m.minScore,
			m.session,
			m.trackDomains,
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
