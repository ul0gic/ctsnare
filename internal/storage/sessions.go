package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// ClearAll removes all hits from the database.
func (d *DB) ClearAll(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM hits")
	if err != nil {
		return fmt.Errorf("clearing all hits: %w", err)
	}
	return nil
}

// ClearSession removes all hits matching the given session tag.
func (d *DB) ClearSession(ctx context.Context, session string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM hits WHERE session = ?", session)
	if err != nil {
		return fmt.Errorf("clearing session %q: %w", session, err)
	}
	return nil
}

// Stats returns aggregate statistics about stored hits.
func (d *DB) Stats(ctx context.Context) (domain.DBStats, error) {
	var stats domain.DBStats

	// Total count.
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hits").Scan(&stats.TotalHits)
	if err != nil {
		return stats, fmt.Errorf("counting total hits: %w", err)
	}

	if stats.TotalHits == 0 {
		stats.BySeverity = make(map[domain.Severity]int)
		return stats, nil
	}

	// Count by severity.
	stats.BySeverity = make(map[domain.Severity]int)
	rows, err := d.db.QueryContext(ctx, "SELECT severity, COUNT(*) FROM hits GROUP BY severity")
	if err != nil {
		return stats, fmt.Errorf("counting by severity: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return stats, fmt.Errorf("scanning severity count: %w", err)
		}
		stats.BySeverity[domain.Severity(severity)] = count
	}
	if err := rows.Err(); err != nil {
		return stats, fmt.Errorf("iterating severity rows: %w", err)
	}

	// First and last hit timestamps. SQLite stores these as text strings,
	// so we scan as strings and parse manually.
	var firstStr, lastStr string
	err = d.db.QueryRowContext(ctx,
		"SELECT MIN(created_at), MAX(created_at) FROM hits",
	).Scan(&firstStr, &lastStr)
	if err != nil {
		return stats, fmt.Errorf("querying hit time range: %w", err)
	}
	stats.FirstHit = parseTimestamp(firstStr)
	stats.LastHit = parseTimestamp(lastStr)

	// Top 10 keywords by occurrence count.
	stats.TopKeywords, err = d.topKeywords(ctx, 10)
	if err != nil {
		return stats, fmt.Errorf("querying top keywords: %w", err)
	}

	return stats, nil
}

// topKeywords parses the keywords JSON arrays from all hits, counts
// occurrences, and returns the top N keywords.
func (d *DB) topKeywords(ctx context.Context, limit int) ([]domain.KeywordCount, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT keywords FROM hits")
	if err != nil {
		return nil, fmt.Errorf("querying keywords: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var keywordsJSON string
		if err := rows.Scan(&keywordsJSON); err != nil {
			return nil, fmt.Errorf("scanning keywords JSON: %w", err)
		}
		var keywords []string
		if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
			continue
		}
		for _, kw := range keywords {
			counts[kw]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating keyword rows: %w", err)
	}

	return topN(counts, limit), nil
}

// topN returns the top N entries from a frequency map, sorted by count descending.
func topN(counts map[string]int, n int) []domain.KeywordCount {
	type kv struct {
		key   string
		count int
	}

	// Collect into slice for sorting.
	items := make([]kv, 0, len(counts))
	for k, v := range counts {
		items = append(items, kv{k, v})
	}

	// Simple selection sort for small N -- typically 10 items.
	for i := 0; i < len(items) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(items); j++ {
			if items[j].count > items[maxIdx].count {
				maxIdx = j
			}
		}
		items[i], items[maxIdx] = items[maxIdx], items[i]
	}

	limit := n
	if limit > len(items) {
		limit = len(items)
	}

	result := make([]domain.KeywordCount, limit)
	for i := 0; i < limit; i++ {
		result[i] = domain.KeywordCount{
			Keyword: items[i].key,
			Count:   items[i].count,
		}
	}

	// Ensure we never return nil (always return empty slice for consistency).
	if result == nil {
		return []domain.KeywordCount{}
	}

	return result
}

// parseTimestamp attempts to parse a timestamp string returned by SQLite.
// Returns zero time if the string is empty or unparseable.
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try common SQLite timestamp formats.
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// Compile-time assertion to catch interface drift early. This verifies
// that *DB satisfies domain.Store without creating an actual instance.
var _ domain.Store = (*DB)(nil)
