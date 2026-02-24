package storage

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDB(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func testHit(domainName string, score int, severity domain.Severity) domain.Hit {
	return domain.Hit{
		Domain:        domainName,
		Score:         score,
		Severity:      severity,
		Keywords:      []string{"bitcoin", "wallet"},
		Issuer:        "Let's Encrypt",
		IssuerCN:      "R3",
		SANDomains:    []string{domainName, "www." + domainName},
		CertNotBefore: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CTLog:         "Google Argon",
		Profile:       "crypto",
		Session:       "session-1",
	}
}

func TestNewDB_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")
	db, err := NewDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
}

func TestInsertAndQuery_Roundtrip(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	hit := testHit("evil-bitcoin.xyz", 6, domain.SeverityHigh)
	err := db.InsertHit(ctx, hit)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, hits, 1)

	got := hits[0]
	assert.Equal(t, hit.Domain, got.Domain)
	assert.Equal(t, hit.Score, got.Score)
	assert.Equal(t, hit.Severity, got.Severity)
	assert.Equal(t, hit.Keywords, got.Keywords)
	assert.Equal(t, hit.Issuer, got.Issuer)
	assert.Equal(t, hit.IssuerCN, got.IssuerCN)
	assert.Equal(t, hit.SANDomains, got.SANDomains)
	assert.Equal(t, hit.CTLog, got.CTLog)
	assert.Equal(t, hit.Profile, got.Profile)
	assert.Equal(t, hit.Session, got.Session)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestUpsert_UpdatesExistingDomain(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	hit1 := testHit("evil-bitcoin.xyz", 4, domain.SeverityMed)
	err := db.UpsertHit(ctx, hit1)
	require.NoError(t, err)

	// Upsert same domain with higher score.
	hit2 := testHit("evil-bitcoin.xyz", 8, domain.SeverityHigh)
	hit2.Keywords = []string{"bitcoin", "wallet", "exchange"}
	err = db.UpsertHit(ctx, hit2)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1, "upsert should not create duplicate rows")

	assert.Equal(t, 8, hits[0].Score)
	assert.Equal(t, domain.SeverityHigh, hits[0].Severity)
	assert.Equal(t, []string{"bitcoin", "wallet", "exchange"}, hits[0].Keywords)
}

func TestQueryHits_KeywordFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("bitcoin-shop.xyz", 4, domain.SeverityMed)
	h1.Keywords = []string{"bitcoin"}
	h2 := testHit("login-page.xyz", 2, domain.SeverityLow)
	h2.Keywords = []string{"login"}

	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Keyword: "bitcoin"})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "bitcoin-shop.xyz", hits[0].Domain)
}

func TestQueryHits_ScoreMinFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("low.com", 2, domain.SeverityLow)))
	require.NoError(t, db.InsertHit(ctx, testHit("high.com", 8, domain.SeverityHigh)))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{ScoreMin: 5})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "high.com", hits[0].Domain)
}

func TestQueryHits_SeverityFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("low.com", 2, domain.SeverityLow)))
	require.NoError(t, db.InsertHit(ctx, testHit("high.com", 8, domain.SeverityHigh)))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Severity: "HIGH"})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "high.com", hits[0].Domain)
}

func TestQueryHits_SessionFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("a.com", 4, domain.SeverityMed)
	h1.Session = "session-a"
	h2 := testHit("b.com", 4, domain.SeverityMed)
	h2.Session = "session-b"

	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Session: "session-a"})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "a.com", hits[0].Domain)
}

func TestQueryHits_SortOrder(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("low.com", 2, domain.SeverityLow)))
	require.NoError(t, db.InsertHit(ctx, testHit("high.com", 8, domain.SeverityHigh)))

	// Sort by score ascending.
	hits, err := db.QueryHits(ctx, domain.QueryFilter{
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, "low.com", hits[0].Domain)
	assert.Equal(t, "high.com", hits[1].Domain)

	// Sort by score descending.
	hits, err = db.QueryHits(ctx, domain.QueryFilter{
		SortBy:  "score",
		SortDir: "DESC",
	})
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, "high.com", hits[0].Domain)
}

func TestQueryHits_LimitOffset(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		h := testHit("domain"+string(rune('a'+i))+".com", i+1, domain.SeverityLow)
		require.NoError(t, db.InsertHit(ctx, h))
	}

	hits, err := db.QueryHits(ctx, domain.QueryFilter{
		Limit:   2,
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	assert.Len(t, hits, 2)

	hits, err = db.QueryHits(ctx, domain.QueryFilter{
		Limit:   2,
		Offset:  2,
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	assert.Len(t, hits, 2)
}

func TestClearAll(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("a.com", 4, domain.SeverityMed)))
	require.NoError(t, db.InsertHit(ctx, testHit("b.com", 4, domain.SeverityMed)))

	err := db.ClearAll(ctx)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Empty(t, hits)
}

func TestClearSession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("a.com", 4, domain.SeverityMed)
	h1.Session = "keep"
	h2 := testHit("b.com", 4, domain.SeverityMed)
	h2.Session = "remove"

	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))

	err := db.ClearSession(ctx, "remove")
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "a.com", hits[0].Domain)
}

func TestStats(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("a.com", 8, domain.SeverityHigh)
	h1.Keywords = []string{"bitcoin", "wallet"}
	h2 := testHit("b.com", 4, domain.SeverityMed)
	h2.Keywords = []string{"bitcoin", "login"}
	h3 := testHit("c.com", 2, domain.SeverityLow)
	h3.Keywords = []string{"wallet"}

	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))
	require.NoError(t, db.InsertHit(ctx, h3))

	stats, err := db.Stats(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.TotalHits)
	assert.Equal(t, 1, stats.BySeverity[domain.SeverityHigh])
	assert.Equal(t, 1, stats.BySeverity[domain.SeverityMed])
	assert.Equal(t, 1, stats.BySeverity[domain.SeverityLow])
	assert.False(t, stats.FirstHit.IsZero())
	assert.False(t, stats.LastHit.IsZero())

	// bitcoin appears in 2 hits, wallet in 2 hits, login in 1 hit.
	assert.NotEmpty(t, stats.TopKeywords)
	kwMap := make(map[string]int)
	for _, kc := range stats.TopKeywords {
		kwMap[kc.Keyword] = kc.Count
	}
	assert.Equal(t, 2, kwMap["bitcoin"])
	assert.Equal(t, 2, kwMap["wallet"])
	assert.Equal(t, 1, kwMap["login"])
}

func TestStats_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	stats, err := db.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalHits)
	assert.NotNil(t, stats.BySeverity)
}

func TestExportJSONL(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("a.com", 4, domain.SeverityMed)))
	require.NoError(t, db.InsertHit(ctx, testHit("b.com", 6, domain.SeverityHigh)))

	var buf bytes.Buffer
	err := db.ExportJSONL(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	// Verify each line is valid JSON.
	for _, line := range lines {
		var hit domain.Hit
		err := json.Unmarshal([]byte(line), &hit)
		assert.NoError(t, err)
		assert.NotEmpty(t, hit.Domain)
	}
}

func TestExportCSV(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("a.com", 4, domain.SeverityMed)))

	var buf bytes.Buffer
	err := db.ExportCSV(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2, "should have header + 1 data row")

	header := lines[0]
	assert.Contains(t, header, "domain")
	assert.Contains(t, header, "score")
	assert.Contains(t, header, "severity")
}

func TestQueryHits_TLDFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("evil.xyz", 4, domain.SeverityMed)))
	require.NoError(t, db.InsertHit(ctx, testHit("good.com", 2, domain.SeverityLow)))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{TLD: "xyz"})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "evil.xyz", hits[0].Domain)
}

func TestSanitizeSortColumn(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"score", "score"},
		{"domain", "domain"},
		{"SCORE", "score"},
		{"invalid_column", "created_at"},
		{"'; DROP TABLE hits; --", "created_at"},
	}

	for _, tt := range tests {
		result := sanitizeSortColumn(tt.input)
		assert.Equal(t, tt.expected, result, "sanitizeSortColumn(%q)", tt.input)
	}
}

// --- Edge case tests (4.1.4) ---

func TestConcurrentReads(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Seed data first (serial writes).
	const totalHits = 20
	for i := 0; i < totalHits; i++ {
		d := fmt.Sprintf("concurrent-%02d.com", i)
		h := testHit(d, i+1, domain.SeverityLow)
		require.NoError(t, db.InsertHit(ctx, h))
	}

	// Concurrent readers should not interfere with each other (WAL mode).
	var wg sync.WaitGroup
	const readers = 5
	const queriesPerReader = 10

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < queriesPerReader; i++ {
				hits, err := db.QueryHits(ctx, domain.QueryFilter{Limit: 10})
				assert.NoError(t, err, "concurrent read should not fail")
				assert.NotEmpty(t, hits)
			}
		}()
	}

	// Also run stats concurrently.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < queriesPerReader; i++ {
			stats, err := db.Stats(ctx)
			assert.NoError(t, err, "concurrent stats should not fail")
			assert.Equal(t, totalHits, stats.TotalHits)
		}
	}()

	wg.Wait()
}

func TestInsertHit_VeryLongDomainName(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	longDomain := strings.Repeat("a", 250) + ".com"
	h := testHit(longDomain, 4, domain.SeverityMed)

	err := db.InsertHit(ctx, h)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, longDomain, hits[0].Domain)
}

func TestInsertHit_UnicodeDomain(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tests := []string{
		"xn--nxasmq6b.com", // punycode
		"\u0431\u0438\u0442\u043a\u043e\u0439\u043d.com", // cyrillic "bitcoin" equivalent
		"\u4e2d\u6587\u57df\u540d.com",                   // Chinese characters
		"caf\u00e9-bitcoin.com",                          // accented characters
	}

	for i, d := range tests {
		h := testHit(d, 4, domain.SeverityMed)
		h.Keywords = []string{"test"}
		err := db.InsertHit(ctx, h)
		require.NoError(t, err, "inserting unicode domain #%d: %q", i, d)
	}

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Len(t, hits, len(tests))
}

func TestInsertHit_EmptyKeywordsArray(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("empty-kw.com", 2, domain.SeverityLow)
	h.Keywords = []string{}
	h.SANDomains = []string{}

	err := db.InsertHit(ctx, h)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Empty(t, hits[0].Keywords)
	assert.Empty(t, hits[0].SANDomains)
}

func TestInsertHit_NilKeywordsSlice(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("nil-kw.com", 2, domain.SeverityLow)
	h.Keywords = nil
	h.SANDomains = nil

	err := db.InsertHit(ctx, h)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	// json.Marshal(nil) produces "null", which json.Unmarshal reads as nil slice.
	// The query should still succeed.
}

func TestInsertHit_EmptyStringFields(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := domain.Hit{
		Domain:        "empty-fields.com",
		Score:         1,
		Severity:      domain.SeverityLow,
		Keywords:      []string{},
		Issuer:        "",
		IssuerCN:      "",
		SANDomains:    []string{},
		CertNotBefore: time.Time{},
		CTLog:         "",
		Profile:       "",
		Session:       "",
	}

	err := db.InsertHit(ctx, h)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "empty-fields.com", hits[0].Domain)
}

func TestQueryHits_AllFiltersSimultaneously(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("target.xyz", 8, domain.SeverityHigh)
	h.Session = "sess-target"
	h.Keywords = []string{"bitcoin", "wallet"}
	require.NoError(t, db.InsertHit(ctx, h))

	// Decoy that should be filtered out by severity.
	h2 := testHit("decoy.xyz", 2, domain.SeverityLow)
	h2.Session = "sess-target"
	h2.Keywords = []string{"bitcoin"}
	require.NoError(t, db.InsertHit(ctx, h2))

	// Decoy that should be filtered out by TLD.
	h3 := testHit("other.com", 8, domain.SeverityHigh)
	h3.Session = "sess-target"
	h3.Keywords = []string{"bitcoin", "wallet"}
	require.NoError(t, db.InsertHit(ctx, h3))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{
		Keyword:  "bitcoin",
		ScoreMin: 5,
		Severity: "HIGH",
		Session:  "sess-target",
		TLD:      "xyz",
		Limit:    10,
		Offset:   0,
		SortBy:   "score",
		SortDir:  "DESC",
	})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "target.xyz", hits[0].Domain)
}

func TestQueryHits_PaginationCoverage(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Insert 10 hits with sequential scores.
	for i := 1; i <= 10; i++ {
		h := testHit(fmt.Sprintf("page-%02d.com", i), i, domain.SeverityLow)
		require.NoError(t, db.InsertHit(ctx, h))
	}

	// Page 1: first 3.
	page1, err := db.QueryHits(ctx, domain.QueryFilter{
		Limit:   3,
		Offset:  0,
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	require.Len(t, page1, 3)
	assert.Equal(t, 1, page1[0].Score)
	assert.Equal(t, 3, page1[2].Score)

	// Page 2: next 3.
	page2, err := db.QueryHits(ctx, domain.QueryFilter{
		Limit:   3,
		Offset:  3,
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	require.Len(t, page2, 3)
	assert.Equal(t, 4, page2[0].Score)
	assert.Equal(t, 6, page2[2].Score)

	// Last page: offset past all records.
	empty, err := db.QueryHits(ctx, domain.QueryFilter{
		Limit:   3,
		Offset:  100,
		SortBy:  "score",
		SortDir: "ASC",
	})
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestQueryHits_SortByEachColumn(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("aaa.com", 2, domain.SeverityLow)
	h1.Session = "sess-a"
	h1.CTLog = "log-a"
	h1.Profile = "crypto"
	require.NoError(t, db.InsertHit(ctx, h1))

	h2 := testHit("zzz.com", 8, domain.SeverityHigh)
	h2.Session = "sess-z"
	h2.CTLog = "log-z"
	h2.Profile = "phishing"
	require.NoError(t, db.InsertHit(ctx, h2))

	sortableColumns := []string{
		"domain", "score", "severity", "session",
		"created_at", "updated_at", "ct_log", "profile",
	}

	for _, col := range sortableColumns {
		hits, err := db.QueryHits(ctx, domain.QueryFilter{
			SortBy:  col,
			SortDir: "ASC",
		})
		require.NoError(t, err, "sort by %s ASC should not error", col)
		assert.Len(t, hits, 2, "sort by %s should return all hits", col)

		hits, err = db.QueryHits(ctx, domain.QueryFilter{
			SortBy:  col,
			SortDir: "DESC",
		})
		require.NoError(t, err, "sort by %s DESC should not error", col)
		assert.Len(t, hits, 2, "sort by %s should return all hits", col)
	}
}

func TestInsertHit_DuplicateDomainReturnsError(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("dup.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	err := db.InsertHit(ctx, h)
	assert.Error(t, err, "inserting duplicate domain should fail")
}

func TestQueryHits_EmptyFilter_ReturnsAll(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("a.com", 2, domain.SeverityLow)))
	require.NoError(t, db.InsertHit(ctx, testHit("b.com", 4, domain.SeverityMed)))
	require.NoError(t, db.InsertHit(ctx, testHit("c.com", 8, domain.SeverityHigh)))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Len(t, hits, 3)
}

func TestClearSession_NonexistentSession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("a.com", 4, domain.SeverityMed)))

	err := db.ClearSession(ctx, "nonexistent")
	require.NoError(t, err, "clearing nonexistent session should not error")

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Len(t, hits, 1, "existing hits should be untouched")
}

func TestClearAll_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	err := db.ClearAll(ctx)
	require.NoError(t, err, "clearing empty DB should not error")
}

func TestQueryHits_TLDFilter_WithDotPrefix(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("evil.xyz", 4, domain.SeverityMed)))
	require.NoError(t, db.InsertHit(ctx, testHit("good.com", 2, domain.SeverityLow)))

	// TLD filter with leading dot should also work.
	hits, err := db.QueryHits(ctx, domain.QueryFilter{TLD: ".xyz"})
	require.NoError(t, err)
	assert.Len(t, hits, 1)
	assert.Equal(t, "evil.xyz", hits[0].Domain)
}

func TestParseTimestamp_Formats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		empty bool
	}{
		{"ISO 8601 with T and Z", "2026-01-15T12:30:00Z", false},
		{"ISO 8601 with space", "2026-01-15 12:30:00", false},
		{"empty string", "", true},
		{"garbage", "not-a-timestamp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestamp(tt.input)
			if tt.empty {
				assert.True(t, result.IsZero())
			} else {
				assert.False(t, result.IsZero())
			}
		})
	}
}

func TestSanitizeSortColumn_AllAllowedColumns(t *testing.T) {
	allowed := []string{
		"domain", "score", "severity", "session",
		"created_at", "updated_at", "ct_log", "profile",
	}

	for _, col := range allowed {
		result := sanitizeSortColumn(col)
		assert.Equal(t, col, result, "allowed column %q should map to itself", col)
	}
}

func TestSanitizeSortColumn_SQLInjectionAttempts(t *testing.T) {
	attacks := []string{
		"score; DROP TABLE hits",
		"score--",
		"1 OR 1=1",
		"' UNION SELECT * FROM sqlite_master--",
		"score\x00",
	}

	for _, attack := range attacks {
		result := sanitizeSortColumn(attack)
		assert.Equal(t, "created_at", result,
			"injection attempt %q should fall back to created_at", attack)
	}
}

// --- Phase 7.1 tests: enrichment, bookmark, delete ---

func TestMigrationV2_NewColumnsExist(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Insert a hit and verify all new columns have their zero-value defaults.
	h := testHit("migration-test.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)

	got := hits[0]
	assert.False(t, got.IsLive, "is_live should default to false")
	assert.Empty(t, got.ResolvedIPs, "resolved_ips should default to empty")
	assert.Empty(t, got.HostingProvider, "hosting_provider should default to empty")
	assert.Equal(t, 0, got.HTTPStatus, "http_status should default to 0")
	assert.True(t, got.LiveCheckedAt.IsZero(), "live_checked_at should default to zero time")
	assert.False(t, got.Bookmarked, "bookmarked should default to false")
}

func TestMigrationV2_Idempotent(t *testing.T) {
	// Opening a second connection to the same DB should not fail
	// because migrations are idempotent.
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "idempotent.db")

	db1, err := NewDB(dbPath)
	require.NoError(t, err)
	db1.Close()

	// Reopen -- migration runs again, should not error.
	db2, err := NewDB(dbPath)
	require.NoError(t, err)
	defer db2.Close()
}

func TestSetBookmark(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("bookmark-me.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	// Bookmark the hit.
	err := db.SetBookmark(ctx, "bookmark-me.com", true)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.True(t, hits[0].Bookmarked, "hit should be bookmarked after SetBookmark(true)")

	// Remove bookmark.
	err = db.SetBookmark(ctx, "bookmark-me.com", false)
	require.NoError(t, err)

	hits, err = db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.False(t, hits[0].Bookmarked, "hit should not be bookmarked after SetBookmark(false)")
}

func TestQueryHits_BookmarkedFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("bookmarked.com", 6, domain.SeverityHigh)
	h2 := testHit("not-bookmarked.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))

	require.NoError(t, db.SetBookmark(ctx, "bookmarked.com", true))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Bookmarked: true})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "bookmarked.com", hits[0].Domain)
}

func TestQueryHits_LiveOnlyFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("live.com", 6, domain.SeverityHigh)
	h2 := testHit("dead.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h1))
	require.NoError(t, db.InsertHit(ctx, h2))

	// Mark one as live via enrichment.
	require.NoError(t, db.UpdateEnrichment(ctx, "live.com", true, []string{"1.2.3.4"}, "cloudflare", 200))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{LiveOnly: true})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "live.com", hits[0].Domain)
}

func TestDeleteHit(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("delete-me.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	err := db.DeleteHit(ctx, "delete-me.com")
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Empty(t, hits, "hit should be gone after DeleteHit")
}

func TestDeleteHit_Nonexistent(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Deleting a domain that doesn't exist should not error.
	err := db.DeleteHit(ctx, "nonexistent.com")
	require.NoError(t, err)
}

func TestDeleteHits_Batch(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		h := testHit(fmt.Sprintf("batch-%d.com", i), i+1, domain.SeverityLow)
		require.NoError(t, db.InsertHit(ctx, h))
	}

	// Delete 3 of the 5 hits.
	err := db.DeleteHits(ctx, []string{"batch-0.com", "batch-2.com", "batch-4.com"})
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{SortBy: "domain", SortDir: "ASC"})
	require.NoError(t, err)
	require.Len(t, hits, 2, "should have 2 remaining hits")
	assert.Equal(t, "batch-1.com", hits[0].Domain)
	assert.Equal(t, "batch-3.com", hits[1].Domain)
}

func TestDeleteHits_EmptySlice(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.InsertHit(ctx, testHit("keep.com", 4, domain.SeverityMed)))

	// Deleting with empty slice should be a no-op.
	err := db.DeleteHits(ctx, []string{})
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	assert.Len(t, hits, 1, "no hits should be deleted")
}

func TestUpdateEnrichment_Roundtrip(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("enrich-me.com", 6, domain.SeverityHigh)
	require.NoError(t, db.InsertHit(ctx, h))

	ips := []string{"104.16.0.1", "104.16.0.2", "2606:4700::1"}
	err := db.UpdateEnrichment(ctx, "enrich-me.com", true, ips, "cloudflare", 200)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)

	got := hits[0]
	assert.True(t, got.IsLive, "should be marked live")
	assert.Equal(t, ips, got.ResolvedIPs, "resolved IPs should roundtrip")
	assert.Equal(t, "cloudflare", got.HostingProvider)
	assert.Equal(t, 200, got.HTTPStatus)
	assert.False(t, got.LiveCheckedAt.IsZero(), "live_checked_at should be set")
}

func TestUpdateEnrichment_NilIPs(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("no-ips.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	err := db.UpdateEnrichment(ctx, "no-ips.com", false, nil, "unknown", 0)
	require.NoError(t, err)

	hits, err := db.QueryHits(ctx, domain.QueryFilter{})
	require.NoError(t, err)
	require.Len(t, hits, 1)

	got := hits[0]
	assert.False(t, got.IsLive)
	assert.Equal(t, "unknown", got.HostingProvider)
	assert.Equal(t, 0, got.HTTPStatus)
}

func TestSanitizeSortColumn_NewColumns(t *testing.T) {
	newCols := []string{"is_live", "bookmarked", "http_status", "live_checked_at"}
	for _, col := range newCols {
		result := sanitizeSortColumn(col)
		assert.Equal(t, col, result, "new column %q should be allowed", col)
	}
}

// --- Phase 7.2 tests: export enrichment roundtrip ---

func TestExportJSONL_EnrichmentFields(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("enriched.com", 6, domain.SeverityHigh)
	require.NoError(t, db.InsertHit(ctx, h))

	// Add enrichment data.
	ips := []string{"104.16.0.1", "104.16.0.2"}
	require.NoError(t, db.UpdateEnrichment(ctx, "enriched.com", true, ips, "cloudflare", 200))

	// Bookmark it.
	require.NoError(t, db.SetBookmark(ctx, "enriched.com", true))

	var buf bytes.Buffer
	err := db.ExportJSONL(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1)

	var exported domain.Hit
	err = json.Unmarshal([]byte(lines[0]), &exported)
	require.NoError(t, err)

	assert.Equal(t, "enriched.com", exported.Domain)
	assert.True(t, exported.IsLive, "JSONL should include is_live=true")
	assert.Equal(t, ips, exported.ResolvedIPs, "JSONL should include resolved IPs")
	assert.Equal(t, "cloudflare", exported.HostingProvider, "JSONL should include hosting provider")
	assert.Equal(t, 200, exported.HTTPStatus, "JSONL should include http status")
	assert.False(t, exported.LiveCheckedAt.IsZero(), "JSONL should include live_checked_at")
	assert.True(t, exported.Bookmarked, "JSONL should include bookmarked=true")
}

func TestExportCSV_EnrichmentFields(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("enriched-csv.com", 6, domain.SeverityHigh)
	require.NoError(t, db.InsertHit(ctx, h))

	ips := []string{"104.16.0.1", "2606:4700::1"}
	require.NoError(t, db.UpdateEnrichment(ctx, "enriched-csv.com", true, ips, "cloudflare", 200))
	require.NoError(t, db.SetBookmark(ctx, "enriched-csv.com", true))

	var buf bytes.Buffer
	err := db.ExportCSV(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2, "should have header + 1 data row")

	header := lines[0]
	assert.Contains(t, header, "is_live")
	assert.Contains(t, header, "resolved_ips")
	assert.Contains(t, header, "hosting_provider")
	assert.Contains(t, header, "http_status")
	assert.Contains(t, header, "live_checked_at")
	assert.Contains(t, header, "bookmarked")

	data := lines[1]
	assert.Contains(t, data, "true", "CSV should contain is_live=true")
	assert.Contains(t, data, "104.16.0.1;2606:4700::1", "CSV should contain semicolon-joined IPs")
	assert.Contains(t, data, "cloudflare", "CSV should contain hosting provider")
	assert.Contains(t, data, "200", "CSV should contain HTTP status")
}

func TestExportJSONL_ZeroEnrichmentFields(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Insert a hit with no enrichment data (zero values).
	h := testHit("plain.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	var buf bytes.Buffer
	err := db.ExportJSONL(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1)

	var exported domain.Hit
	err = json.Unmarshal([]byte(lines[0]), &exported)
	require.NoError(t, err)

	assert.Equal(t, "plain.com", exported.Domain)
	assert.False(t, exported.IsLive, "zero-value enrichment: is_live should be false")
	assert.Empty(t, exported.ResolvedIPs, "zero-value enrichment: resolved_ips should be empty")
	assert.Empty(t, exported.HostingProvider, "zero-value enrichment: hosting_provider should be empty")
	assert.Equal(t, 0, exported.HTTPStatus, "zero-value enrichment: http_status should be 0")
	assert.True(t, exported.LiveCheckedAt.IsZero(), "zero-value enrichment: live_checked_at should be zero")
	assert.False(t, exported.Bookmarked, "zero-value enrichment: bookmarked should be false")
}

func TestExportCSV_ZeroEnrichmentFields(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h := testHit("plain-csv.com", 4, domain.SeverityMed)
	require.NoError(t, db.InsertHit(ctx, h))

	var buf bytes.Buffer
	err := db.ExportCSV(ctx, &buf, domain.QueryFilter{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	// Parse the CSV to verify the new columns have zero values.
	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2, "header + 1 row")

	header := records[0]
	row := records[1]

	// Find column indexes.
	colIdx := make(map[string]int)
	for i, name := range header {
		colIdx[name] = i
	}

	assert.Equal(t, "false", row[colIdx["is_live"]], "zero-value is_live should be 'false'")
	assert.Empty(t, row[colIdx["resolved_ips"]], "zero-value resolved_ips should be empty")
	assert.Empty(t, row[colIdx["hosting_provider"]], "zero-value hosting_provider should be empty")
	assert.Equal(t, "0", row[colIdx["http_status"]], "zero-value http_status should be '0'")
	assert.Empty(t, row[colIdx["live_checked_at"]], "zero-value live_checked_at should be empty")
	assert.Equal(t, "false", row[colIdx["bookmarked"]], "zero-value bookmarked should be 'false'")
}
