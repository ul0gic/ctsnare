package cmd

import (
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dbPath  string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "ctsnare",
	Short: "Monitor Certificate Transparency logs for suspicious domains",
	Long: `ctsnare is a real-time Certificate Transparency (CT) log monitor
that scores newly issued TLS certificates against keyword profiles to
detect phishing, typosquatting, and brand impersonation.

It polls public CT logs, extracts domain names from certificates, scores
them using configurable heuristic profiles, and stores hits in a local
SQLite database. A terminal UI provides a live feed and historical
exploration of flagged domains.`,
	PersistentPreRunE: initLogging,
	Version:           resolveVersion(),
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database path override")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug logging")

	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}
}

// initLogging sets the global slog logger from --verbose: JSON/Debug when set,
// otherwise discarded so log lines don't corrupt the TUI.
func initLogging(_ *cobra.Command, _ []string) error {
	var handler slog.Handler

	if verbose {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
