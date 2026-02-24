package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
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
