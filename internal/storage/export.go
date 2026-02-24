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

	// Header row. New enrichment and bookmark columns are appended at the end
	// for backward compatibility with parsers that use column names.
	header := []string{
		"domain", "score", "severity", "keywords", "issuer", "issuer_cn",
		"san_domains", "cert_not_before", "ct_log", "profile", "session",
		"created_at", "updated_at",
		"is_live", "resolved_ips", "hosting_provider", "http_status",
		"live_checked_at", "bookmarked",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, hit := range hits {
		isLive := "false"
		if hit.IsLive {
			isLive = "true"
		}
		bookmarked := "false"
		if hit.Bookmarked {
			bookmarked = "true"
		}
		liveCheckedAt := ""
		if !hit.LiveCheckedAt.IsZero() {
			liveCheckedAt = hit.LiveCheckedAt.UTC().Format("2006-01-02T15:04:05Z")
		}

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
			isLive,
			strings.Join(hit.ResolvedIPs, ";"),
			hit.HostingProvider,
			strconv.Itoa(hit.HTTPStatus),
			liveCheckedAt,
			bookmarked,
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}
