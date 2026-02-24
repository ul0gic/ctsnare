package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
		_, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%s\n",
			hit.Severity, hit.Score, hit.Domain, kw, issuer, ts)
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
	header := []string{"severity", "score", "domain", "keywords", "issuer", "issuer_cn", "ct_log", "profile", "session", "timestamp"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, hit := range hits {
		row := []string{
			string(hit.Severity),
			fmt.Sprintf("%d", hit.Score),
			hit.Domain,
			strings.Join(hit.Keywords, ";"),
			hit.Issuer,
			hit.IssuerCN,
			hit.CTLog,
			hit.Profile,
			hit.Session,
			hit.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// FormatStats writes database statistics in a human-readable format.
func FormatStats(stats domain.DBStats, w io.Writer) error {
	_, err := fmt.Fprintf(w, "Database Statistics\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "==================\n\n")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Total Hits:  %d\n\n", stats.TotalHits)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "By Severity:\n")
	if err != nil {
		return err
	}
	for _, sev := range []domain.Severity{domain.SeverityHigh, domain.SeverityMed, domain.SeverityLow} {
		count := stats.BySeverity[sev]
		_, err = fmt.Fprintf(w, "  %-6s %d\n", sev, count)
		if err != nil {
			return err
		}
	}

	if len(stats.TopKeywords) > 0 {
		_, err = fmt.Fprintf(w, "\nTop Keywords:\n")
		if err != nil {
			return err
		}
		for i, kw := range stats.TopKeywords {
			_, err = fmt.Fprintf(w, "  %2d. %-20s %d\n", i+1, kw.Keyword, kw.Count)
			if err != nil {
				return err
			}
		}
	}

	if !stats.FirstHit.IsZero() {
		_, err = fmt.Fprintf(w, "\nDate Range:\n")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "  First Hit: %s\n", stats.FirstHit.Format("2006-01-02 15:04:05"))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "  Last Hit:  %s\n", stats.LastHit.Format("2006-01-02 15:04:05"))
		if err != nil {
			return err
		}
	}

	return nil
}
