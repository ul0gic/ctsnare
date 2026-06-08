package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// FormatTable writes hits as a formatted ASCII table.
func FormatTable(hits []domain.Hit, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, err := fmt.Fprintln(tw, "SEVERITY\tSCORE\tDOMAIN\tKEYWORDS\tISSUER\tTIMESTAMP")
	if err != nil {
		return fmt.Errorf("writing table header: %w", err)
	}

	for _, hit := range hits {
		kw := strings.Join(hit.Keywords, ", ")
		if len(kw) > 30 {
			kw = kw[:27] + "..."
		}
		issuer := hit.IssuerCN
		if len(issuer) > 25 {
			issuer = issuer[:22] + "..."
		}
		ts := hit.CreatedAt.Format("2006-01-02 15:04:05")

		// Bookmark and live indicators.
		domainStr := hit.Domain
		if hit.Bookmarked {
			domainStr = "* " + domainStr
		}
		if hit.IsLive {
			domainStr += " [L]"
		}

		_, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%s\n",
			hit.Severity, hit.Score, domainStr, kw, issuer, ts)
		if err != nil {
			return fmt.Errorf("writing table row: %w", err)
		}
	}
	return tw.Flush()
}

// FormatJSON writes hits as one JSON object per line (JSONL).
func FormatJSON(hits []domain.Hit, w io.Writer) error {
	enc := json.NewEncoder(w)
	for _, hit := range hits {
		if err := enc.Encode(hit); err != nil {
			return fmt.Errorf("encoding hit as JSON: %w", err)
		}
	}
	return nil
}

// FormatCSV writes hits as CSV with a header row.
func FormatCSV(hits []domain.Hit, w io.Writer) error {
	cw := csv.NewWriter(w)
	header := []string{
		"severity", "score", "domain", "keywords", "issuer", "issuer_cn",
		"ct_log", "profile", "session", "timestamp",
		"is_live", "resolved_ips", "hosting_provider", "http_status", "bookmarked",
		"base_domain",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, hit := range hits {
		row := []string{
			string(hit.Severity),
			strconv.Itoa(hit.Score),
			hit.Domain,
			strings.Join(hit.Keywords, ";"),
			hit.Issuer,
			hit.IssuerCN,
			hit.CTLog,
			hit.Profile,
			hit.Session,
			hit.CreatedAt.Format("2006-01-02T15:04:05Z"),
			strconv.FormatBool(hit.IsLive),
			strings.Join(hit.ResolvedIPs, ";"),
			hit.HostingProvider,
			strconv.Itoa(hit.HTTPStatus),
			strconv.FormatBool(hit.Bookmarked),
			hit.BaseDomain,
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// FormatStats writes database statistics in a human-readable format.
// errWriter wraps an io.Writer and records the first write error, so a sequence
// of formatted writes can be expressed without an error check after each call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func FormatStats(stats domain.DBStats, w io.Writer) error {
	ew := &errWriter{w: w}

	ew.printf("Database Statistics\n")
	ew.printf("==================\n\n")
	ew.printf("Total Hits:  %d\n\n", stats.TotalHits)

	ew.printf("By Severity:\n")
	for _, sev := range []domain.Severity{domain.SeverityHigh, domain.SeverityMed, domain.SeverityLow} {
		ew.printf("  %-6s %d\n", sev, stats.BySeverity[sev])
	}

	if len(stats.TopKeywords) > 0 {
		ew.printf("\nTop Keywords:\n")
		for i, kw := range stats.TopKeywords {
			ew.printf("  %2d. %-20s %d\n", i+1, kw.Keyword, kw.Count)
		}
	}

	if !stats.FirstHit.IsZero() {
		ew.printf("\nDate Range:\n")
		ew.printf("  First Hit: %s\n", stats.FirstHit.Format("2006-01-02 15:04:05"))
		ew.printf("  Last Hit:  %s\n", stats.LastHit.Format("2006-01-02 15:04:05"))
	}

	return ew.err
}
