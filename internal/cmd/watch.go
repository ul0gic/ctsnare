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
	"github.com/ul0gic/ctsnare/internal/domainutil"
	"github.com/ul0gic/ctsnare/internal/enrichment"
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
	watchBacktrack    int64
	watchMinScore     int
	watchDomains      []string
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

Domain-tracker mode: pass one or more --domain flags to store EVERY newly
issued certificate for a target apex and all its subdomains, regardless of
score or keywords. In this mode --min-score and keyword-profile gating have no
effect. Pair with --session to tag the run, and --backtrack to catch recent
issuance at startup.

Examples:
  ctsnare watch
  ctsnare watch --profile crypto --session morning-run
  ctsnare watch --headless --poll-interval 10s
  ctsnare watch --backtrack 1000
  ctsnare watch --domain openai.com --domain anthropic.com --session brands
  ctsnare watch --domain openai.com --backtrack 50000 --session openai`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVar(&watchProfile, "profile", "all", `keyword profile to use for scoring (built-ins: "crypto", "phishing", "all")`)
	watchCmd.Flags().StringVar(&watchSession, "session", "", "optional tag to group hits from this run (queryable later with --session)")
	watchCmd.Flags().BoolVar(&watchHeadless, "headless", false, "run without TUI — poll and store only (for servers and background use)")
	watchCmd.Flags().IntVar(&watchBatchSize, "batch-size", 0, "number of CT log entries to fetch per poll (default: 256 from config)")
	watchCmd.Flags().DurationVar(&watchPollInterval, "poll-interval", 0, "wait time between polls per log (default: 5s from config)")
	watchCmd.Flags().Int64Var(&watchBacktrack, "backtrack", 0, "start N entries behind the current log tip for immediate results (default: 0, start at tip)")
	watchCmd.Flags().IntVar(&watchMinScore, "min-score", 0, "minimum score to store a hit (default: 0, store all scored hits)")
	watchCmd.Flags().StringArrayVar(&watchDomains, "domain", nil, "track an exact apex + all its subdomains, storing every matching cert (repeatable; enables tracker mode, ignores --min-score)")

	rootCmd.AddCommand(watchCmd)
}

// runWatch wires config, storage, scoring, profiles, and pollers, then
// launches either the TUI dashboard or headless polling loop.
func runWatch(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, watchBatchSize, watchPollInterval, watchBacktrack, watchMinScore)

	slog.Info("config loaded",
		"db_path", cfg.DBPath,
		"batch_size", cfg.BatchSize,
		"poll_interval", cfg.PollInterval,
		"ct_logs", len(cfg.CTLogs))

	store, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer closeStore(store)

	burnerTLDs, cheapTLDs := config.ResolveTLDTiers(cfg.TLDTiers)
	scorer := scoring.NewEngine(scoring.Config{
		BurnerTLDs:       burnerTLDs,
		CheapTLDs:        cheapTLDs,
		WatchPlatforms:   profile.WatchPlatformSuffixes,
		CategoryKeywords: profile.CategoryKeywords(),
	})

	profileMgr := profile.NewManager(cfg.CustomProfiles)
	prof, err := profileMgr.LoadProfile(watchProfile)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}

	// The merged globals+overrides list replaces the profile's default suffixes.
	prof.SkipSuffixes = config.MergeSkipSuffixes(profile.GlobalSkipSuffixes, cfg.SkipOverrides)

	slog.Info("profile loaded",
		"name", prof.Name,
		"keywords", len(prof.Keywords),
		"effective_skip_suffixes", len(prof.SkipSuffixes))

	// Normalize any --domain targets. When non-empty, pollers run in
	// tracker mode and store every matching cert regardless of score.
	trackDomains := domainutil.NormalizeTrackTargets(watchDomains)
	if len(trackDomains) > 0 {
		slog.Info("domain-tracker mode enabled",
			"targets", trackDomains,
			"note", "--min-score and keyword gating ignored; storing all matching certs")
	}

	hitChan := make(chan domain.Hit, 256)
	pollerStatsChan := make(chan poller.PollStats, 64)

	pollerMgr := poller.NewManager(cfg, scorer, store, prof, cfg.Backtrack, cfg.MinScore, watchSession, trackDomains)

	// Discard channel streams zero-scored domain names for the TUI activity feed.
	discardChan := make(chan string, 256)

	if watchHeadless {
		return runHeadless(store, pollerMgr, hitChan, pollerStatsChan, discardChan)
	}
	return runTUI(store, pollerMgr, hitChan, pollerStatsChan, discardChan, prof.Name)
}

// runHeadless starts pollers and enrichment without a TUI, blocking until SIGINT/SIGTERM.
func runHeadless(
	store *storage.DB,
	pollerMgr *poller.Manager,
	hitChan chan domain.Hit,
	statsChan chan poller.PollStats,
	discardChan chan string,
) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting headless mode")

	if err := pollerMgr.Start(ctx, hitChan, statsChan, discardChan); err != nil {
		return fmt.Errorf("starting pollers: %w", err)
	}

	enrichResultChan := make(chan enrichment.EnrichResult, 256)
	enricher := enrichment.NewEnricher(store, enrichResultChan)
	go func() {
		if err := enricher.Run(ctx); err != nil {
			slog.Debug("enrichment pipeline stopped", "error", err)
		}
	}()

	go func() {
		for hit := range hitChan {
			enricher.Enqueue(hit.Domain)
		}
	}()
	// Drain stats, enrichment result, and discard channels so pollers never block.
	go func() {
		for range statsChan {
		}
	}()
	go func() {
		for range enrichResultChan {
		}
	}()
	go func() {
		for range discardChan {
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, stopping pollers")

	pollerMgr.Stop()
	close(hitChan)
	close(statsChan)
	close(discardChan)

	slog.Info("headless mode shutdown complete")
	return nil
}

// runTUI starts pollers, the enrichment pipeline, and the Bubble Tea TUI dashboard.
func runTUI(
	store *storage.DB,
	pollerMgr *poller.Manager,
	hitChan chan domain.Hit,
	pollerStatsChan chan poller.PollStats,
	discardChan chan string,
	profileName string,
) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The poller emits per-log stats; the TUI expects them aggregated.
	tuiStatsChan := make(chan tui.PollStats, 64)
	go bridgePollerStats(ctx, pollerStatsChan, tuiStatsChan)

	if err := pollerMgr.Start(ctx, hitChan, pollerStatsChan, discardChan); err != nil {
		return fmt.Errorf("starting pollers: %w", err)
	}

	enrichResultChan := make(chan enrichment.EnrichResult, 256)
	enricher := enrichment.NewEnricher(store, enrichResultChan)
	go func() {
		if err := enricher.Run(ctx); err != nil {
			slog.Debug("enrichment pipeline stopped", "error", err)
		}
	}()

	tuiHitChan := make(chan domain.Hit, 256)
	go tapHits(ctx, hitChan, tuiHitChan, enricher)

	app := tui.NewApp(store, tuiHitChan, tuiStatsChan, enrichResultChan, discardChan, profileName)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		cancel()
		pollerMgr.Stop()
		return fmt.Errorf("running TUI: %w", err)
	}

	slog.Info("TUI exited, shutting down pollers")
	cancel()
	pollerMgr.Stop()
	close(hitChan)
	close(pollerStatsChan)
	close(discardChan)

	slog.Info("watch command shutdown complete")
	return nil
}

// tapHits enqueues each hit for enrichment and forwards it to dst, never
// blocking — a full dst buffer drops the TUI copy. Closes dst on src/ctx done.
func tapHits(ctx context.Context, src <-chan domain.Hit, dst chan<- domain.Hit, enricher *enrichment.Enricher) {
	defer close(dst)
	for {
		select {
		case <-ctx.Done():
			return
		case hit, ok := <-src:
			if !ok {
				return
			}
			enricher.Enqueue(hit.Domain)
			select {
			case dst <- hit:
			default:
			}
		}
	}
}

// bridgePollerStats recalculates the aggregate tui.PollStats on every per-log
// update and forwards it on the TUI channel.
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
			agg := aggregatePollStats(perLog, time.Since(startTime))

			select {
			case out <- agg:
			default:
				// Don't block if TUI is slow to consume.
			}
		}
	}
}

// aggregatePollStats sums per-log poller stats into a single tui.PollStats and
// derives the session-average certs/sec and hits/min rates over elapsed.
func aggregatePollStats(perLog map[string]poller.PollStats, elapsed time.Duration) tui.PollStats {
	var totalCerts, totalHits int64
	for _, s := range perLog {
		totalCerts += s.CertsScanned
		totalHits += s.HitsFound
	}

	var certsPerSec, hitsPerMin float64
	if secs := elapsed.Seconds(); secs > 0 {
		certsPerSec = float64(totalCerts) / secs
	}
	if mins := elapsed.Minutes(); mins > 0 {
		hitsPerMin = float64(totalHits) / mins
	}

	return tui.PollStats{
		CertsScanned: totalCerts,
		HitsFound:    totalHits,
		CertsPerSec:  certsPerSec,
		HitsPerMin:   hitsPerMin,
		ActiveLogs:   len(perLog),
	}
}
