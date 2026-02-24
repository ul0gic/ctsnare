package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/poller"
	"github.com/ul0gic/ctsnare/internal/profile"
	"github.com/ul0gic/ctsnare/internal/scoring"
	"github.com/ul0gic/ctsnare/internal/storage"
	"github.com/ul0gic/ctsnare/internal/tui"
)

var (
	watchProfile      string
	watchSession      string
	watchHeadless     bool
	watchBatchSize    int
	watchPollInterval time.Duration
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start live CT log monitoring",
	Long: `Start monitoring Certificate Transparency logs in real-time.

Polls public CT logs, scores new certificates against the selected
keyword profile, and stores hits in the local database.

By default, starts the interactive TUI dashboard. Use --headless
for non-interactive mode (polling and storage only, suitable for
servers and background processes).

Examples:
  ctsnare watch
  ctsnare watch --profile crypto --session morning-run
  ctsnare watch --headless --poll-interval 10s`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVar(&watchProfile, "profile", "all", `keyword profile to use for scoring (built-ins: "crypto", "phishing", "all")`)
	watchCmd.Flags().StringVar(&watchSession, "session", "", "optional tag to group hits from this run (queryable later with --session)")
	watchCmd.Flags().BoolVar(&watchHeadless, "headless", false, "run without TUI — poll and store only (for servers and background use)")
	watchCmd.Flags().IntVar(&watchBatchSize, "batch-size", 0, "number of CT log entries to fetch per poll (default: 256 from config)")
	watchCmd.Flags().DurationVar(&watchPollInterval, "poll-interval", 0, "wait time between polls per log (default: 5s from config)")

	rootCmd.AddCommand(watchCmd)
}

// runWatch wires config, storage, scoring, profiles, and pollers, then
// launches either the TUI dashboard or headless polling loop.
func runWatch(_ *cobra.Command, _ []string) error {
	// Load configuration and apply flag overrides.
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, watchBatchSize, watchPollInterval)

	slog.Info("config loaded",
		"db_path", cfg.DBPath,
		"batch_size", cfg.BatchSize,
		"poll_interval", cfg.PollInterval,
		"ct_logs", len(cfg.CTLogs))

	// Open storage.
	store, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer store.Close()

	// Create scoring engine.
	scorer := scoring.NewEngine()

	// Load keyword profile.
	profileMgr := profile.NewManager(cfg.CustomProfiles)
	prof, err := profileMgr.LoadProfile(watchProfile)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}

	slog.Info("profile loaded", "name", prof.Name, "keywords", len(prof.Keywords))

	// Create channels for hit and stats streaming.
	hitChan := make(chan domain.Hit, 256)
	pollerStatsChan := make(chan poller.PollStats, 64)

	// Create poller manager.
	pollerMgr := poller.NewManager(cfg, scorer, store, prof)

	if watchHeadless {
		return runHeadless(pollerMgr, hitChan, pollerStatsChan)
	}
	return runTUI(store, pollerMgr, hitChan, pollerStatsChan, prof.Name)
}

// runHeadless starts pollers without a TUI, blocking until SIGINT/SIGTERM.
func runHeadless(
	pollerMgr *poller.Manager,
	hitChan chan domain.Hit,
	statsChan chan poller.PollStats,
) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting headless mode")

	if err := pollerMgr.Start(ctx, hitChan, statsChan); err != nil {
		return fmt.Errorf("starting pollers: %w", err)
	}

	// Drain hit and stats channels in background so pollers never block.
	go func() {
		for range hitChan {
		}
	}()
	go func() {
		for range statsChan {
		}
	}()

	// Block until context is cancelled by signal.
	<-ctx.Done()
	slog.Info("shutdown signal received, stopping pollers")

	pollerMgr.Stop()
	close(hitChan)
	close(statsChan)

	slog.Info("headless mode shutdown complete")
	return nil
}

// runTUI starts pollers and the Bubble Tea TUI dashboard.
func runTUI(
	store *storage.DB,
	pollerMgr *poller.Manager,
	hitChan chan domain.Hit,
	pollerStatsChan chan poller.PollStats,
	profileName string,
) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Bridge poller stats to TUI stats in a separate goroutine.
	// The poller emits per-log stats; the TUI expects aggregated stats.
	tuiStatsChan := make(chan tui.PollStats, 64)
	go bridgePollerStats(ctx, pollerStatsChan, tuiStatsChan)

	if err := pollerMgr.Start(ctx, hitChan, pollerStatsChan); err != nil {
		return fmt.Errorf("starting pollers: %w", err)
	}

	// Create TUI app with real store and channels.
	app := tui.NewApp(store, hitChan, tuiStatsChan, profileName)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseAllMotion())

	// Run TUI -- blocks until user quits.
	if _, err := p.Run(); err != nil {
		cancel()
		pollerMgr.Stop()
		return fmt.Errorf("running TUI: %w", err)
	}

	// Graceful shutdown: cancel context, stop pollers, close channels.
	slog.Info("TUI exited, shutting down pollers")
	cancel()
	pollerMgr.Stop()
	close(hitChan)
	close(pollerStatsChan)

	slog.Info("watch command shutdown complete")
	return nil
}

// bridgePollerStats aggregates per-log poller.PollStats into tui.PollStats
// and forwards them on the TUI channel. Each per-log update recalculates
// the aggregate view.
func bridgePollerStats(
	ctx context.Context,
	in <-chan poller.PollStats,
	out chan<- tui.PollStats,
) {
	defer close(out)

	perLog := make(map[string]poller.PollStats)
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case stats, ok := <-in:
			if !ok {
				return
			}
			perLog[stats.LogName] = stats

			// Aggregate across all logs.
			var totalCerts int64
			var totalHits int64
			for _, s := range perLog {
				totalCerts += s.CertsScanned
				totalHits += s.HitsFound
			}

			elapsed := time.Since(startTime).Seconds()
			var certsPerSec float64
			if elapsed > 0 {
				certsPerSec = float64(totalCerts) / elapsed
			}

			agg := tui.PollStats{
				CertsScanned: totalCerts,
				HitsFound:    totalHits,
				CertsPerSec:  certsPerSec,
				ActiveLogs:   len(perLog),
			}

			select {
			case out <- agg:
			default:
				// Don't block if TUI is slow to consume.
			}
		}
	}
}
