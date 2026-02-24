package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
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
for non-interactive mode (polling and storage only).`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVar(&watchProfile, "profile", "all", "keyword profile to use for scoring")
	watchCmd.Flags().StringVar(&watchSession, "session", "", "session tag for grouping hits")
	watchCmd.Flags().BoolVar(&watchHeadless, "headless", false, "run without TUI (poll and store only)")
	watchCmd.Flags().IntVar(&watchBatchSize, "batch-size", 0, "override batch size from config")
	watchCmd.Flags().DurationVar(&watchPollInterval, "poll-interval", 0, "override poll interval from config")

	rootCmd.AddCommand(watchCmd)
}

// runWatch is the placeholder RunE for the watch command.
// Real component wiring (config, store, scorer, profile, poller, TUI) happens in Phase 3.
func runWatch(_ *cobra.Command, _ []string) error {
	// Phase 3 will wire: config loading, storage, scoring engine, profile manager,
	// poller manager, hit/stats channels, and either TUI or headless loop.
	return fmt.Errorf("watch command not yet wired — integration happens in Phase 3")
}
