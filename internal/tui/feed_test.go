package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func makeHit(domainName string, score int, severity domain.Severity) domain.Hit {
	return domain.Hit{
		Domain:    domainName,
		Score:     score,
		Severity:  severity,
		Keywords:  []string{"test"},
		IssuerCN:  "TestCA",
		CreatedAt: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
	}
}

func initFeedModel(t *testing.T, profile string, width, height int) FeedModel {
	t.Helper()
	m := NewFeedModel(profile)
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

func TestNewFeedModel_InitialState(t *testing.T) {
	m := NewFeedModel("all")

	assert.Empty(t, m.hits, "hits should start empty")
	assert.Empty(t, m.topKeywords, "top keywords should start empty")
	assert.Equal(t, "all", m.profile)
	assert.False(t, m.ready, "should not be ready before window size")
}

func TestFeedModel_Init_ReturnsNil(t *testing.T) {
	m := NewFeedModel("all")
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestFeedModel_WindowSizeMsg_SetsReady(t *testing.T) {
	m := NewFeedModel("all")
	assert.False(t, m.ready)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.True(t, m.ready)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestFeedModel_ViewBeforeReady(t *testing.T) {
	m := NewFeedModel("all")
	view := m.View()
	assert.Equal(t, "Initializing feed...", view)
}

func TestFeedModel_HitMsg_PrependsHit(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	hit1 := makeHit("first.com", 4, domain.SeverityMed)
	m, _ = m.Update(HitMsg{Hit: hit1})
	require.Len(t, m.hits, 1)
	assert.Equal(t, "first.com", m.hits[0].Domain)

	hit2 := makeHit("second.com", 6, domain.SeverityHigh)
	m, _ = m.Update(HitMsg{Hit: hit2})
	require.Len(t, m.hits, 2)
	assert.Equal(t, "second.com", m.hits[0].Domain, "new hit should be first (prepended)")
	assert.Equal(t, "first.com", m.hits[1].Domain)
}

func TestFeedModel_HitBuffer_MaxSize(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	// Fill past the max.
	for i := 0; i < maxFeedHits+50; i++ {
		hit := makeHit("domain-"+strings.Repeat("x", 5)+".com", 2, domain.SeverityLow)
		hit.Domain = "hit-" + time.Now().Add(time.Duration(i)*time.Second).Format("150405") + ".com"
		m, _ = m.Update(HitMsg{Hit: hit})
	}

	assert.LessOrEqual(t, len(m.hits), maxFeedHits,
		"hit buffer should not exceed maxFeedHits (%d)", maxFeedHits)
}

func TestFeedModel_StatsMsg_UpdatesStats(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	stats := PollStats{
		CertsScanned: 1000,
		HitsFound:    50,
		CertsPerSec:  33.5,
		ActiveLogs:   3,
	}
	m, _ = m.Update(StatsMsg{Stats: stats})

	assert.Equal(t, int64(1000), m.stats.CertsScanned)
	assert.Equal(t, int64(50), m.stats.HitsFound)
	assert.InDelta(t, 33.5, m.stats.CertsPerSec, 0.01)
	assert.Equal(t, 3, m.stats.ActiveLogs)
}

func TestFeedModel_View_ContainsHitDomain(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	hit := makeHit("evil-phishing.xyz", 8, domain.SeverityHigh)
	m, _ = m.Update(HitMsg{Hit: hit})

	view := m.View()
	assert.Contains(t, view, "evil-phishing.xyz")
}

func TestFeedModel_View_ContainsStatusBar(t *testing.T) {
	m := initFeedModel(t, "crypto", 120, 40)

	m, _ = m.Update(StatsMsg{Stats: PollStats{
		CertsScanned: 500,
		HitsFound:    10,
		ActiveLogs:   2,
	}})

	view := m.View()
	assert.Contains(t, view, "Scanned: 500")
	assert.Contains(t, view, "Hits: 10")
	assert.Contains(t, view, "crypto")
}

func TestFeedModel_View_ContainsHeader(t *testing.T) {
	m := initFeedModel(t, "all", 120, 40)
	view := m.View()
	assert.Contains(t, view, "Live Feed")
}

func TestFeedModel_View_EmptyHitsShowsWaiting(t *testing.T) {
	m := initFeedModel(t, "all", 120, 40)
	view := m.View()
	assert.Contains(t, view, "Waiting for hits...")
}

func TestFeedModel_WindowResize_UpdatesViewport(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	// Resize to different dimensions.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestFeedModel_SeverityStyling(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	highHit := makeHit("high-sev.xyz", 8, domain.SeverityHigh)
	m, _ = m.Update(HitMsg{Hit: highHit})

	view := m.View()
	assert.Contains(t, view, "HIGH")
}

func TestFeedModel_KeywordCountsUpdate(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)

	hit1 := makeHit("btc1.com", 4, domain.SeverityMed)
	hit1.Keywords = []string{"bitcoin", "wallet"}
	m, _ = m.Update(HitMsg{Hit: hit1})

	hit2 := makeHit("btc2.com", 4, domain.SeverityMed)
	hit2.Keywords = []string{"bitcoin", "login"}
	m, _ = m.Update(HitMsg{Hit: hit2})

	// bitcoin should appear with count 2.
	found := false
	for _, kc := range m.topKeywords {
		if kc.Keyword == "bitcoin" {
			assert.Equal(t, 2, kc.Count)
			found = true
			break
		}
	}
	assert.True(t, found, "bitcoin should be in top keywords")
}

func TestFeedModel_ContentWidthNarrow(t *testing.T) {
	m := initFeedModel(t, "test", 80, 40)
	// Below keywordSidebarMin (100), content width should be full width.
	assert.Equal(t, 80, m.contentWidth())
}

func TestFeedModel_ContentWidthWide(t *testing.T) {
	m := initFeedModel(t, "test", 120, 40)
	// Above keywordSidebarMin, sidebar takes 26 chars.
	assert.Equal(t, 120-26, m.contentWidth())
}

func TestFeedModel_MinimumContentHeight(t *testing.T) {
	// Very small terminal height -- content height should be clamped to 1.
	m := initFeedModel(t, "test", 120, 3)
	assert.True(t, m.ready)
}

// --- Helper function tests ---

func TestPrependHit_Basic(t *testing.T) {
	hits := []domain.Hit{
		makeHit("existing.com", 2, domain.SeverityLow),
	}

	newHit := makeHit("new.com", 4, domain.SeverityMed)
	result := prependHit(hits, newHit, 10)

	require.Len(t, result, 2)
	assert.Equal(t, "new.com", result[0].Domain)
	assert.Equal(t, "existing.com", result[1].Domain)
}

func TestPrependHit_EnforcesMaxSize(t *testing.T) {
	var hits []domain.Hit
	for i := 0; i < 5; i++ {
		hits = append(hits, makeHit("old.com", 1, domain.SeverityLow))
	}

	newHit := makeHit("new.com", 4, domain.SeverityMed)
	result := prependHit(hits, newHit, 5)

	assert.Len(t, result, 5, "should not exceed max size")
	assert.Equal(t, "new.com", result[0].Domain, "new hit should be first")
}

func TestPrependHit_EmptySlice(t *testing.T) {
	result := prependHit(nil, makeHit("first.com", 2, domain.SeverityLow), 10)
	require.Len(t, result, 1)
	assert.Equal(t, "first.com", result[0].Domain)
}

func TestUpdateKeywordCounts_Basic(t *testing.T) {
	counts := updateKeywordCounts(nil, []string{"bitcoin", "wallet"})

	assert.Len(t, counts, 2)
	m := make(map[string]int)
	for _, kc := range counts {
		m[kc.Keyword] = kc.Count
	}
	assert.Equal(t, 1, m["bitcoin"])
	assert.Equal(t, 1, m["wallet"])
}

func TestUpdateKeywordCounts_Accumulates(t *testing.T) {
	counts := []domain.KeywordCount{
		{Keyword: "bitcoin", Count: 5},
		{Keyword: "wallet", Count: 3},
	}

	result := updateKeywordCounts(counts, []string{"bitcoin", "login"})

	m := make(map[string]int)
	for _, kc := range result {
		m[kc.Keyword] = kc.Count
	}
	assert.Equal(t, 6, m["bitcoin"])
	assert.Equal(t, 3, m["wallet"])
	assert.Equal(t, 1, m["login"])
}

func TestUpdateKeywordCounts_SortedDescending(t *testing.T) {
	counts := updateKeywordCounts(nil, []string{"a", "b", "a", "c", "a", "b"})

	require.NotEmpty(t, counts)
	assert.Equal(t, "a", counts[0].Keyword, "most frequent keyword should be first")
	assert.Equal(t, 3, counts[0].Count)
}

func TestUpdateKeywordCounts_CapsAtTopKeywordsCount(t *testing.T) {
	// Generate more unique keywords than topKeywordsCount.
	keywords := make([]string, topKeywordsCount+5)
	for i := range keywords {
		keywords[i] = "keyword-" + time.Now().Add(time.Duration(i)*time.Second).Format("150405")
	}

	counts := updateKeywordCounts(nil, keywords)
	assert.LessOrEqual(t, len(counts), topKeywordsCount)
}

func TestUpdateKeywordCounts_EmptyKeywords(t *testing.T) {
	existing := []domain.KeywordCount{
		{Keyword: "bitcoin", Count: 5},
	}

	result := updateKeywordCounts(existing, []string{})
	assert.Len(t, result, 1)
	assert.Equal(t, "bitcoin", result[0].Keyword)
	assert.Equal(t, 5, result[0].Count)
}

func TestRenderSeverityTag(t *testing.T) {
	tests := []struct {
		severity string
		contains string
	}{
		{"HIGH", "HIGH"},
		{"MED", "MED"},
		{"LOW", "LOW"},
		{"", ""},
	}

	for _, tt := range tests {
		result := renderSeverityTag(tt.severity)
		assert.Contains(t, result, tt.contains,
			"renderSeverityTag(%q) should contain %q", tt.severity, tt.contains)
	}
}
