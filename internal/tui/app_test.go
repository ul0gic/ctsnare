package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func TestNewApp(t *testing.T) {
	app := NewApp(nil, nil, nil, "all")
	if app.activeView != viewFeed {
		t.Errorf("expected initial view to be feed (%d), got %d", viewFeed, app.activeView)
	}
}

func TestAppViewSwitchingTab(t *testing.T) {
	app := NewApp(nil, nil, nil, "all")
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
	app := NewApp(nil, hitCh, nil, "test")

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
	app := NewApp(nil, nil, nil, "all")
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
	app := NewApp(nil, nil, nil, "all")
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
	app := NewApp(nil, nil, nil, "all")
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
	app := NewApp(nil, nil, nil, "all")
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
