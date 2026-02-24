package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/storage"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long: `Manage the local SQLite database: view stats, clear data, export, or show the database path.

Examples:
  ctsnare db stats
  ctsnare db clear --confirm
  ctsnare db export --format csv --output hits.csv
  ctsnare db path`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

// db stats subcommand
var dbStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long:  `Display aggregate statistics about stored hits: total count, by severity, top keywords, and date range.`,
	RunE:  runDBStats,
}

// db clear subcommand
var (
	dbClearConfirm bool
	dbClearSession string
)

var dbClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear stored hits from the database",
	Long: `Delete hits from the database.

Requires --confirm to prevent accidental deletion. Without --session,
all hits are removed. With --session, only hits from that session are removed.

Examples:
  ctsnare db clear --confirm
  ctsnare db clear --session morning-run --confirm`,
	RunE: runDBClear,
}

// db export subcommand
var (
	dbExportFormat string
	dbExportOutput string
)

var dbExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export hits to JSONL or CSV",
	Long: `Export all stored hits to a file or stdout in JSONL or CSV format.

JSONL (default) outputs one JSON object per line. CSV includes a header row.
If --output is not specified, output is written to stdout.

Examples:
  ctsnare db export
  ctsnare db export --format csv --output hits.csv
  ctsnare db export --format jsonl | jq '.domain'`,
	RunE: runDBExport,
}

// db path subcommand
var dbPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the database file path",
	Long:  `Print the path to the SQLite database file.`,
	RunE:  runDBPath,
}

func init() {
	dbClearCmd.Flags().BoolVar(&dbClearConfirm, "confirm", false, "required: confirm deletion to prevent accidents")
	dbClearCmd.Flags().StringVar(&dbClearSession, "session", "", "only clear hits tagged with this session name")

	dbExportCmd.Flags().StringVar(&dbExportFormat, "format", "jsonl", "export format: jsonl (default) or csv")
	dbExportCmd.Flags().StringVar(&dbExportOutput, "output", "", "write to this file path instead of stdout")

	dbCmd.AddCommand(dbStatsCmd)
	dbCmd.AddCommand(dbClearCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbPathCmd)
	rootCmd.AddCommand(dbCmd)
}

// openDB loads config and opens the database, returning a cleanup function.
func openDB() (*storage.DB, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, 0, 0)

	if _, statErr := os.Stat(cfg.DBPath); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("database not found at %s — run 'ctsnare watch' first to start collecting hits", cfg.DBPath)
	}

	store, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return store, nil
}

// runDBStats displays aggregate statistics about stored hits.
func runDBStats(_ *cobra.Command, _ []string) error {
	store, err := openDB()
	if err != nil {
		return err
	}
	defer store.Close()

	stats, err := store.Stats(context.Background())
	if err != nil {
		return fmt.Errorf("getting stats: %w", err)
	}

	return FormatStats(stats, os.Stdout)
}

// runDBClear deletes hits from the database.
func runDBClear(_ *cobra.Command, _ []string) error {
	if !dbClearConfirm {
		return fmt.Errorf("use --confirm to confirm deletion")
	}

	store, err := openDB()
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()

	if dbClearSession != "" {
		if err := store.ClearSession(ctx, dbClearSession); err != nil {
			return fmt.Errorf("clearing session %q: %w", dbClearSession, err)
		}
		fmt.Fprintf(os.Stderr, "Cleared all hits for session %q.\n", dbClearSession)
		return nil
	}

	if err := store.ClearAll(ctx); err != nil {
		return fmt.Errorf("clearing database: %w", err)
	}
	fmt.Fprintln(os.Stderr, "All hits cleared from database.")
	return nil
}

// runDBExport exports hits to JSONL or CSV.
func runDBExport(_ *cobra.Command, _ []string) error {
	store, err := openDB()
	if err != nil {
		return err
	}
	defer store.Close()

	// Determine output destination.
	var w *os.File
	if dbExportOutput != "" {
		w, err = os.Create(dbExportOutput)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	ctx := context.Background()
	filter := domain.QueryFilter{}

	switch dbExportFormat {
	case "csv":
		if err := store.ExportCSV(ctx, w, filter); err != nil {
			return fmt.Errorf("exporting CSV: %w", err)
		}
	default:
		if err := store.ExportJSONL(ctx, w, filter); err != nil {
			return fmt.Errorf("exporting JSONL: %w", err)
		}
	}

	if dbExportOutput != "" {
		fmt.Fprintf(os.Stderr, "Exported to %s (%s format).\n", dbExportOutput, dbExportFormat)
	}
	return nil
}

// runDBPath prints the configured database file path.
func runDBPath(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, 0, 0)

	fmt.Println(cfg.DBPath)
	return nil
}
