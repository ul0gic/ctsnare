package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/storage"
)

var (
	queryKeyword    string
	queryScoreMin   int
	querySince      string
	queryTLD        string
	querySession    string
	querySeverity   string
	queryFormat     string
	queryLimit      int
	queryBookmarked bool
	queryLiveOnly   bool
	queryDomain     string
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
  ctsnare query --keyword wallet --severity HIGH --since 24h --format json | jq '.domain'
  ctsnare query --domain openai.com --session openai`,
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryKeyword, "keyword", "", "filter by keyword substring match against matched keywords")
	queryCmd.Flags().IntVar(&queryScoreMin, "min-score", 0, "minimum score (HIGH=8+, MED=5-7, LOW=1-4)")
	queryCmd.Flags().StringVar(&querySince, "since", "", `only show hits from within this duration (e.g., "1h", "24h", "7d")`)
	queryCmd.Flags().StringVar(&queryTLD, "tld", "", `filter by TLD suffix (e.g., ".xyz" or "xyz")`)
	queryCmd.Flags().StringVar(&queryDomain, "domain", "", "filter to an exact apex + all its subdomains (e.g., openai.com matches api.openai.com)")
	queryCmd.Flags().StringVar(&querySession, "session", "", "filter by session tag set with 'ctsnare watch --session'")
	queryCmd.Flags().StringVar(&querySeverity, "severity", "", "filter by severity: HIGH, MED, or LOW")
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format: table (default), json (JSONL), or csv")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 50, "maximum number of results to return")
	queryCmd.Flags().BoolVar(&queryBookmarked, "bookmarked", false, "show only bookmarked hits")
	queryCmd.Flags().BoolVar(&queryLiveOnly, "live-only", false, "show only domains that responded to HTTP liveness probe")

	rootCmd.AddCommand(queryCmd)
}

// runQuery opens the database, queries hits with the given filters, and formats output.
func runQuery(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	config.MergeFlags(cfg, dbPath, 0, 0, 0, 0)

	since, err := parseSince(querySince)
	if err != nil {
		return fmt.Errorf("invalid --since value %q: %w", querySince, err)
	}

	// Check if the database file exists before attempting to open it.
	if _, statErr := os.Stat(cfg.DBPath); os.IsNotExist(statErr) {
		fmt.Fprintln(os.Stderr, "No database found. Run 'ctsnare watch' first to start collecting hits.")
		return nil
	}

	store, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer closeStore(store)

	filter := domain.QueryFilter{
		Keyword:  queryKeyword,
		ScoreMin: queryScoreMin,
		Since:    since,
		TLD:      queryTLD,
		Session:  querySession,
		Severity: querySeverity,
		Domain:   queryDomain,
		Limit:    queryLimit,
		SortBy:   "score",
		SortDir:  "DESC",
		LiveOnly: queryLiveOnly,
	}
	// Only apply the bookmark filter when --bookmarked was explicitly set.
	// --bookmarked / --bookmarked=true → only bookmarked; --bookmarked=false →
	// only non-bookmarked; flag absent → no bookmark filter.
	if cmd.Flags().Changed("bookmarked") {
		filter.Bookmarked = &queryBookmarked
	}

	hits, err := store.QueryHits(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("querying hits: %w", err)
	}

	return WriteQueryOutput(hits, queryFormat)
}

// parseSince parses a duration string, additionally accepting a "d" (day)
// suffix that time.ParseDuration lacks (e.g. "7d" → 168h). An empty string
// means no time filter.
func parseSince(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.ParseFloat(days, 64)
		if err != nil {
			return 0, errors.New(`expected a number before "d"`)
		}
		return time.Duration(n * 24 * float64(time.Hour)), nil
	}
	return time.ParseDuration(s)
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
