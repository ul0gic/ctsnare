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

All flags are optional and composable — unset flags match everything.
Results are sorted by score descending by default.

Examples:
  ctsnare query
  ctsnare query --severity HIGH --format json
  ctsnare query --keyword casino --since 12h
  ctsnare query --keyword wallet --severity HIGH --since 24h --format json | jq '.domain'`,
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryKeyword, "keyword", "", "filter by keyword substring match against matched keywords")
	queryCmd.Flags().IntVar(&queryScoreMin, "score-min", 0, "minimum score (HIGH=6+, MED=4-5, LOW=1-3)")
	queryCmd.Flags().DurationVar(&querySince, "since", 0, `only show hits from within this duration (e.g., "1h", "24h", "7d")`)
	queryCmd.Flags().StringVar(&queryTLD, "tld", "", `filter by TLD suffix (e.g., ".xyz" or "xyz")`)
	queryCmd.Flags().StringVar(&querySession, "session", "", "filter by session tag set with 'ctsnare watch --session'")
	queryCmd.Flags().StringVar(&querySeverity, "severity", "", "filter by severity: HIGH, MED, or LOW")
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format: table (default), json (JSONL), or csv")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 50, "maximum number of results to return (default: 50)")

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
