package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  `Manage the local SQLite database: view stats, clear data, export, or show the database path.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

// db stats subcommand
var (
	dbStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show database statistics",
		Long:  `Display aggregate statistics about stored hits: total count, by severity, top keywords, and date range.`,
		RunE:  runDBStats,
	}
)

// db clear subcommand
var (
	dbClearConfirm bool
	dbClearSession string
)

var dbClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear stored hits from the database",
	Long: `Delete hits from the database. Requires --confirm flag to prevent accidental deletion.
Use --session to clear only hits from a specific session.`,
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
	Long:  `Export all stored hits to a file or stdout in JSONL or CSV format.`,
	RunE:  runDBExport,
}

// db path subcommand
var dbPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the database file path",
	Long:  `Print the path to the SQLite database file.`,
	RunE:  runDBPath,
}

func init() {
	dbClearCmd.Flags().BoolVar(&dbClearConfirm, "confirm", false, "confirm deletion (required)")
	dbClearCmd.Flags().StringVar(&dbClearSession, "session", "", "only clear hits from this session")

	dbExportCmd.Flags().StringVar(&dbExportFormat, "format", "jsonl", "export format: jsonl or csv")
	dbExportCmd.Flags().StringVar(&dbExportOutput, "output", "", "output file path (stdout if empty)")

	dbCmd.AddCommand(dbStatsCmd)
	dbCmd.AddCommand(dbClearCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbPathCmd)
	rootCmd.AddCommand(dbCmd)
}

// runDBStats is the placeholder RunE for the db stats command.
// Real storage wiring happens in Phase 3.
func runDBStats(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("db stats not yet wired -- integration happens in Phase 3")
}

// runDBClear is the placeholder RunE for the db clear command.
// Real storage wiring happens in Phase 3.
func runDBClear(_ *cobra.Command, _ []string) error {
	if !dbClearConfirm {
		return fmt.Errorf("use --confirm to confirm deletion")
	}
	return fmt.Errorf("db clear not yet wired -- integration happens in Phase 3")
}

// runDBExport is the placeholder RunE for the db export command.
// Real storage wiring happens in Phase 3.
func runDBExport(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("db export not yet wired -- integration happens in Phase 3")
}

// runDBPath is the placeholder RunE for the db path command.
// Real storage wiring happens in Phase 3.
func runDBPath(_ *cobra.Command, _ []string) error {
	// Phase 3 will read from config to determine the DB path.
	// For now, show the default XDG-compliant path.
	fmt.Println("~/.local/share/ctsnare/ctsnare.db")
	return nil
}
