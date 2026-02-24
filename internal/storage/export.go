package storage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// ExportJSONL writes one JSON line per hit to the writer, using the given
// filter to select records. The filter's Limit is set to 0 (no limit).
func (d *DB) ExportJSONL(ctx context.Context, w io.Writer, filter domain.QueryFilter) error {
	filter.Limit = 0
	filter.Offset = 0

	hits, err := d.QueryHits(ctx, filter)
	if err != nil {
		return fmt.Errorf("querying hits for JSONL export: %w", err)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, hit := range hits {
		if err := enc.Encode(hit); err != nil {
			return fmt.Errorf("encoding hit to JSONL: %w", err)
		}
	}

	return nil
}

// ExportCSV writes hits as CSV with a header row, using the given filter
// to select records. The filter's Limit is set to 0 (no limit).
func (d *DB) ExportCSV(ctx context.Context, w io.Writer, filter domain.QueryFilter) error {
	filter.Limit = 0
	filter.Offset = 0

	hits, err := d.QueryHits(ctx, filter)
	if err != nil {
		return fmt.Errorf("querying hits for CSV export: %w", err)
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row.
	header := []string{
		"domain", "score", "severity", "keywords", "issuer", "issuer_cn",
		"san_domains", "cert_not_before", "ct_log", "profile", "session",
		"created_at", "updated_at",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, hit := range hits {
		record := []string{
			hit.Domain,
			strconv.Itoa(hit.Score),
			string(hit.Severity),
			strings.Join(hit.Keywords, ";"),
			hit.Issuer,
			hit.IssuerCN,
			strings.Join(hit.SANDomains, ";"),
			hit.CertNotBefore.UTC().Format("2006-01-02T15:04:05Z"),
			hit.CTLog,
			hit.Profile,
			hit.Session,
			hit.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			hit.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}
