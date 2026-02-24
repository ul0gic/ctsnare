package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/storage"
)

var (
	queryKeyword  string
	queryScoreMin int
	querySince    time.Duration
	queryTLD      string
	querySession  string
	querySeverity string
	queryFormat   string
	queryLimit    int
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Search and filter stored hits",
	Long: `Search the local database for stored hits matching the given filters.

Filters can be combined. Results are output in table, JSON, or CSV format.`,
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryKeyword, "keyword", "", "filter by keyword substring")
	queryCmd.Flags().IntVar(&queryScoreMin, "score-min", 0, "minimum score threshold")
	queryCmd.Flags().DurationVar(&querySince, "since", 0, "only hits from this duration ago (e.g., 24h, 7d)")
	queryCmd.Flags().StringVar(&queryTLD, "tld", "", "filter by top-level domain suffix")
	queryCmd.Flags().StringVar(&querySession, "session", "", "filter by session tag")
	queryCmd.Flags().StringVar(&querySeverity, "severity", "", "filter by severity (HIGH, MED, LOW)")
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format: table, json, csv")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 50, "maximum number of results")

	rootCmd.AddCommand(queryCmd)
}

// runQuery opens the database, queries hits with the given filters, and formats output.
func runQuery(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, 0, 0)

	// Check if the database file exists before attempting to open it.
	if _, statErr := os.Stat(cfg.DBPath); os.IsNotExist(statErr) {
		fmt.Fprintln(os.Stderr, "No database found. Run 'ctsnare watch' first to start collecting hits.")
		return nil
	}

	store, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer store.Close()

	filter := domain.QueryFilter{
		Keyword:  queryKeyword,
		ScoreMin: queryScoreMin,
		Since:    querySince,
		TLD:      queryTLD,
		Session:  querySession,
		Severity: querySeverity,
		Limit:    queryLimit,
		SortBy:   "score",
		SortDir:  "DESC",
	}

	hits, err := store.QueryHits(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("querying hits: %w", err)
	}

	return WriteQueryOutput(hits, queryFormat)
}

// WriteQueryOutput writes hits in the requested format to stdout.
func WriteQueryOutput(hits []domain.Hit, format string) error {
	if len(hits) == 0 {
		fmt.Fprintln(os.Stderr, "No hits found matching the given filters.")
		return nil
	}

	switch format {
	case "json":
		return FormatJSON(hits, os.Stdout)
	case "csv":
		return FormatCSV(hits, os.Stdout)
	default:
		return FormatTable(hits, os.Stdout)
	}
}
