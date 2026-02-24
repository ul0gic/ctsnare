package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/domain"
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
// Phase 3 will replace the placeholder store opening with real config + storage wiring.
func runQuery(_ *cobra.Command, _ []string) error {
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

	// Phase 3 will wire: config loading, storage opening, store.QueryHits(ctx, filter).
	// For now, the filter is constructed to validate flag parsing and will return early.
	_ = filter
	fmt.Fprintln(os.Stderr, "query command not yet wired -- integration happens in Phase 3")
	return nil
}

// WriteQueryOutput writes hits in the requested format to stdout.
// Used by the query command RunE after Phase 3 storage wiring.
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
