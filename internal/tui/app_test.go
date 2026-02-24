package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func TestNewApp(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	if app.activeView != viewFeed {
		t.Errorf("expected initial view to be feed (%d), got %d", viewFeed, app.activeView)
	}
}

func TestAppViewSwitchingTab(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	// Provide window size so sub-models are ready
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	if app.activeView != viewFeed {
		t.Fatalf("expected feed view, got %d", app.activeView)
	}

	// Press Tab to switch to explorer
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(AppModel)
	if app.activeView != viewExplorer {
		t.Errorf("expected explorer view after tab, got %d", app.activeView)
	}

	// Press Tab again to switch back to feed
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(AppModel)
	if app.activeView != viewFeed {
		t.Errorf("expected feed view after second tab, got %d", app.activeView)
	}
}

func TestAppHitMsgUpdatesFeed(t *testing.T) {
	hitCh := make(chan domain.Hit, 1)
	app := NewApp(nil, hitCh, nil, nil, nil, "test")

	// Initialize with window size
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	hit := domain.Hit{
		Domain:    "evil-phish.example.com",
		Score:     6,
		Severity:  domain.SeverityHigh,
		Keywords:  []string{"phish", "evil"},
		IssuerCN:  "Let's Encrypt",
		CreatedAt: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
	}

	model, _ = app.Update(HitMsg{Hit: hit})
	app = model.(AppModel)

	if len(app.feed.hits) != 1 {
		t.Errorf("expected 1 hit in feed, got %d", len(app.feed.hits))
	}
	if app.feed.hits[0].Domain != "evil-phish.example.com" {
		t.Errorf("expected domain evil-phish.example.com, got %s", app.feed.hits[0].Domain)
	}
}

func TestAppQuitMessage(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command on ctrl+c, got nil")
	}

	// Verify the command produces a QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppShowDetail(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	hit := domain.Hit{
		Domain:   "test.example.com",
		Score:    4,
		Severity: domain.SeverityMed,
	}

	model, _ = app.Update(ShowDetailMsg{Hit: hit})
	app = model.(AppModel)

	if app.activeView != viewDetail {
		t.Errorf("expected detail view, got %d", app.activeView)
	}
	if app.detail == nil {
		t.Fatal("expected detail model to be set")
	}
	if app.detail.hit.Domain != "test.example.com" {
		t.Errorf("expected detail domain test.example.com, got %s", app.detail.hit.Domain)
	}
}

func TestAppSwitchViewMsg(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Switch to detail first
	hit := domain.Hit{Domain: "test.com", Score: 3, Severity: domain.SeverityLow}
	model, _ = app.Update(ShowDetailMsg{Hit: hit})
	app = model.(AppModel)
	if app.activeView != viewDetail {
		t.Fatalf("expected detail view, got %d", app.activeView)
	}

	// SwitchViewMsg back to explorer should clear detail
	model, _ = app.Update(SwitchViewMsg{View: viewExplorer})
	app = model.(AppModel)
	if app.activeView != viewExplorer {
		t.Errorf("expected explorer view, got %d", app.activeView)
	}
	if app.detail != nil {
		t.Error("expected detail to be nil after switching away")
	}
}

func TestAppFilterOverlay(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "all")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Switch to explorer first
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(AppModel)
	if app.activeView != viewExplorer {
		t.Fatalf("expected explorer view, got %d", app.activeView)
	}

	// Press f to open filter
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app = model.(AppModel)
	if app.activeView != viewFilter {
		t.Errorf("expected filter view, got %d", app.activeView)
	}
	if app.filter == nil {
		t.Fatal("expected filter model to be set")
	}

	// Cancel filter
	model, _ = app.Update(FilterCancelledMsg{})
	app = model.(AppModel)
	if app.activeView != viewExplorer {
		t.Errorf("expected explorer view after cancel, got %d", app.activeView)
	}
	if app.filter != nil {
		t.Error("expected filter to be nil after cancel")
	}
}

// --- Phase 7.4 message handling tests ---

func TestAppEnrichmentMsg_UpdatesFeedHit(t *testing.T) {
	hitCh := make(chan domain.Hit, 1)
	app := NewApp(nil, hitCh, nil, nil, nil, "test")

	// Initialize with window size.
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Add a hit to the feed.
	hit := domain.Hit{
		Domain:   "enrich-target.example.com",
		Score:    6,
		Severity: domain.SeverityHigh,
		Keywords: []string{"phish"},
	}
	model, _ = app.Update(HitMsg{Hit: hit})
	app = model.(AppModel)

	if len(app.feed.hits) != 1 {
		t.Fatalf("expected 1 hit in feed, got %d", len(app.feed.hits))
	}
	// Initially, enrichment fields should be zero values.
	if app.feed.hits[0].IsLive {
		t.Error("hit should not be live before enrichment")
	}

	// Send an EnrichmentMsg for the same domain.
	enrichMsg := EnrichmentMsg{
		Domain:          "enrich-target.example.com",
		IsLive:          true,
		ResolvedIPs:     []string{"104.16.0.1", "104.16.0.2"},
		HostingProvider: "cloudflare",
		HTTPStatus:      200,
	}
	model, _ = app.Update(enrichMsg)
	app = model.(AppModel)

	// Verify the feed hit was updated with enrichment data.
	feedHit := app.feed.hits[0]
	if !feedHit.IsLive {
		t.Error("feed hit should be live after EnrichmentMsg")
	}
	if feedHit.HTTPStatus != 200 {
		t.Errorf("expected http status 200, got %d", feedHit.HTTPStatus)
	}
	if feedHit.HostingProvider != "cloudflare" {
		t.Errorf("expected hosting provider cloudflare, got %s", feedHit.HostingProvider)
	}
	if len(feedHit.ResolvedIPs) != 2 {
		t.Errorf("expected 2 resolved IPs, got %d", len(feedHit.ResolvedIPs))
	}
}

func TestAppEnrichmentMsg_UpdatesExplorerHit(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Populate the explorer's hits directly.
	app.explorer.hits = []domain.Hit{
		{Domain: "explorer-hit.com", Score: 5, Severity: domain.SeverityMed},
	}

	// Send EnrichmentMsg.
	enrichMsg := EnrichmentMsg{
		Domain:          "explorer-hit.com",
		IsLive:          true,
		ResolvedIPs:     []string{"1.2.3.4"},
		HostingProvider: "aws",
		HTTPStatus:      301,
	}
	model, _ = app.Update(enrichMsg)
	app = model.(AppModel)

	// Explorer hit should be updated.
	if !app.explorer.hits[0].IsLive {
		t.Error("explorer hit should be live after EnrichmentMsg")
	}
	if app.explorer.hits[0].HTTPStatus != 301 {
		t.Errorf("expected http status 301, got %d", app.explorer.hits[0].HTTPStatus)
	}
	if app.explorer.hits[0].HostingProvider != "aws" {
		t.Errorf("expected hosting provider aws, got %s", app.explorer.hits[0].HostingProvider)
	}
}

func TestAppEnrichmentMsg_UpdatesDetailView(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Open detail view for a hit.
	hit := domain.Hit{Domain: "detail-hit.com", Score: 7, Severity: domain.SeverityHigh}
	model, _ = app.Update(ShowDetailMsg{Hit: hit})
	app = model.(AppModel)

	if app.activeView != viewDetail {
		t.Fatalf("expected detail view, got %d", app.activeView)
	}
	if app.detail == nil {
		t.Fatal("detail model should be set")
	}

	// Send enrichment for the same domain.
	enrichMsg := EnrichmentMsg{
		Domain:          "detail-hit.com",
		IsLive:          true,
		ResolvedIPs:     []string{"10.0.0.1"},
		HostingProvider: "gcp",
		HTTPStatus:      200,
	}
	model, _ = app.Update(enrichMsg)
	app = model.(AppModel)

	if !app.detail.hit.IsLive {
		t.Error("detail hit should be live after EnrichmentMsg")
	}
	if app.detail.hit.HostingProvider != "gcp" {
		t.Errorf("expected hosting provider gcp, got %s", app.detail.hit.HostingProvider)
	}
}

func TestAppEnrichmentMsg_NonMatchingDomain_NoUpdate(t *testing.T) {
	hitCh := make(chan domain.Hit, 1)
	app := NewApp(nil, hitCh, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Add a hit.
	hit := domain.Hit{Domain: "existing.com", Score: 4, Severity: domain.SeverityMed}
	model, _ = app.Update(HitMsg{Hit: hit})
	app = model.(AppModel)

	// Send enrichment for a different domain.
	enrichMsg := EnrichmentMsg{
		Domain:          "other.com",
		IsLive:          true,
		ResolvedIPs:     []string{"5.5.5.5"},
		HostingProvider: "fastly",
		HTTPStatus:      200,
	}
	model, _ = app.Update(enrichMsg)
	app = model.(AppModel)

	// Original hit should be unchanged.
	if app.feed.hits[0].IsLive {
		t.Error("non-matching enrichment should not update existing hit")
	}
}

func TestAppDiscardedDomainMsg_ForwardedToFeed(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	initialDiscardCount := app.feed.discardCount

	// Send a DiscardedDomainMsg.
	model, _ = app.Update(DiscardedDomainMsg{Domain: "discarded-domain.example.com"})
	app = model.(AppModel)

	// Feed should have received and processed the discard.
	if app.feed.discardCount != initialDiscardCount+1 {
		t.Errorf("expected discard count %d, got %d", initialDiscardCount+1, app.feed.discardCount)
	}
	if len(app.feed.discards) < 1 {
		t.Fatal("expected at least 1 discard entry in feed")
	}
	if app.feed.discards[0].Domain != "discarded-domain.example.com" {
		t.Errorf("expected discarded domain discarded-domain.example.com, got %s", app.feed.discards[0].Domain)
	}
}

func TestAppDiscardedDomainMsg_MultipleDiscards(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Send multiple discards.
	for i := 0; i < 5; i++ {
		model, _ = app.Update(DiscardedDomainMsg{Domain: "discard.com"})
		app = model.(AppModel)
	}

	if app.feed.discardCount != 5 {
		t.Errorf("expected 5 discards, got %d", app.feed.discardCount)
	}
}

func TestAppBookmarkToggleMsg_RefreshesExplorer(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Populate the explorer with some hits.
	app.explorer.hits = []domain.Hit{
		{Domain: "bookmark-test.com", Score: 4, Severity: domain.SeverityMed, Bookmarked: false},
		{Domain: "other.com", Score: 2, Severity: domain.SeverityLow, Bookmarked: false},
	}

	// Send BookmarkToggleMsg.
	model, _ = app.Update(BookmarkToggleMsg{Domain: "bookmark-test.com", Bookmarked: true})
	app = model.(AppModel)

	// The explorer's local hit should be updated.
	if !app.explorer.hits[0].Bookmarked {
		t.Error("explorer hit should be bookmarked after BookmarkToggleMsg")
	}
	// Other hit should be unchanged.
	if app.explorer.hits[1].Bookmarked {
		t.Error("other hit should not be affected by BookmarkToggleMsg")
	}
}

func TestAppDeleteHitsMsg_TriggersExplorerReload(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	// Populate explorer with some hits.
	app.explorer.hits = []domain.Hit{
		{Domain: "keep.com", Score: 4, Severity: domain.SeverityMed},
		{Domain: "delete.com", Score: 2, Severity: domain.SeverityLow},
	}
	app.explorer.selected = map[int]bool{1: true}

	// Send DeleteHitsMsg.
	model, cmd := app.Update(DeleteHitsMsg{Domains: []string{"delete.com"}})
	app = model.(AppModel)

	// After DeleteHitsMsg, the explorer should have cleared selection
	// and started a reload (indicated by loading=true and a non-nil cmd).
	if len(app.explorer.selected) != 0 {
		t.Errorf("expected selection to be cleared after delete, got %d selected", len(app.explorer.selected))
	}
	if !app.explorer.loading {
		t.Error("explorer should be loading after DeleteHitsMsg (reload triggered)")
	}
	if cmd == nil {
		t.Error("expected a reload command after DeleteHitsMsg")
	}
}

func TestAppStatsMsgForwarded(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, nil, "test")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(AppModel)

	stats := PollStats{
		CertsScanned: 5000,
		HitsFound:    100,
		CertsPerSec:  50.0,
		ActiveLogs:   3,
		HitsPerMin:   2.5,
	}

	model, _ = app.Update(StatsMsg{Stats: stats})
	app = model.(AppModel)

	if app.feed.stats.CertsScanned != 5000 {
		t.Errorf("expected certs scanned 5000, got %d", app.feed.stats.CertsScanned)
	}
	if app.feed.stats.HitsFound != 100 {
		t.Errorf("expected hits found 100, got %d", app.feed.stats.HitsFound)
	}
}
