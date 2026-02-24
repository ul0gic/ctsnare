package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	viewFeed     = 0
	viewExplorer = 1
	viewDetail   = 2
	viewFilter   = 3
)

// AppModel is the root Bubble Tea model that manages view switching and message routing.
type AppModel struct {
	activeView int
	feed       FeedModel
	explorer   ExplorerModel
	detail     *DetailModel
	filter     *FilterModel
	keys       KeyMap
	width      int
	height     int
	hitChan    <-chan domain.Hit
	statsChan  <-chan PollStats
}

// NewApp creates a new root TUI application model.
// The store may be nil during Phase 2; real wiring happens in Phase 3.
// The hitChan and statsChan may be nil if the TUI is opened without polling.
func NewApp(store domain.Store, hitChan <-chan domain.Hit, statsChan <-chan PollStats, profile string) AppModel {
	return AppModel{
		activeView: viewFeed,
		feed:       NewFeedModel(profile),
		explorer:   NewExplorerModel(store),
		keys:       DefaultKeyMap(),
		hitChan:    hitChan,
		statsChan:  statsChan,
	}
}

// Init returns the initial commands for the app, including channel subscriptions.
func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.explorer.Init(),
	}
	if m.hitChan != nil {
		cmds = append(cmds, waitForHit(m.hitChan))
	}
	if m.statsChan != nil {
		cmds = append(cmds, waitForStats(m.statsChan))
	}
	return tea.Batch(cmds...)
}

// Update handles all incoming messages and delegates to the active sub-model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.feed, _ = m.feed.Update(msg)
		m.explorer, _ = m.explorer.Update(msg)
		if m.detail != nil {
			*m.detail, _ = m.detail.Update(msg)
		}
		if m.filter != nil {
			*m.filter, _ = m.filter.Update(msg)
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit: ctrl+c always quits, q quits unless in filter overlay
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if key.Matches(msg, m.keys.Quit) && m.activeView != viewFilter {
			return m, tea.Quit
		}

		// Tab toggles between feed and explorer
		if key.Matches(msg, m.keys.Tab) && m.activeView != viewFilter && m.activeView != viewDetail {
			if m.activeView == viewFeed {
				m.activeView = viewExplorer
			} else {
				m.activeView = viewFeed
			}
			return m, nil
		}

		// Filter overlay toggle
		if key.Matches(msg, m.keys.Filter) && m.activeView == viewExplorer {
			f := NewFilterModel()
			f.width = m.width
			f.height = m.height
			m.filter = &f
			m.activeView = viewFilter
			return m, m.filter.Init()
		}

	case HitMsg:
		var cmd tea.Cmd
		m.feed, cmd = m.feed.Update(msg)
		cmds = append(cmds, cmd)
		if m.hitChan != nil {
			cmds = append(cmds, waitForHit(m.hitChan))
		}
		return m, tea.Batch(cmds...)

	case StatsMsg:
		var cmd tea.Cmd
		m.feed, cmd = m.feed.Update(msg)
		cmds = append(cmds, cmd)
		if m.statsChan != nil {
			cmds = append(cmds, waitForStats(m.statsChan))
		}
		return m, tea.Batch(cmds...)

	case HitsLoadedMsg:
		var cmd tea.Cmd
		m.explorer, cmd = m.explorer.Update(msg)
		return m, cmd

	case ShowDetailMsg:
		d := NewDetailModel(msg.Hit)
		d.width = m.width
		d.height = m.height
		m.detail = &d
		m.activeView = viewDetail
		sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		*m.detail, _ = m.detail.Update(sizeMsg)
		return m, nil

	case SwitchViewMsg:
		m.activeView = msg.View
		if msg.View != viewDetail {
			m.detail = nil
		}
		if msg.View != viewFilter {
			m.filter = nil
		}
		return m, nil

	case FilterAppliedMsg:
		m.activeView = viewExplorer
		m.filter = nil
		cmd := m.explorer.SetFilter(msg.Filter)
		return m, cmd

	case FilterCancelledMsg:
		m.activeView = viewExplorer
		m.filter = nil
		return m, nil
	}

	// Delegate to active view
	var cmd tea.Cmd
	switch m.activeView {
	case viewFeed:
		m.feed, cmd = m.feed.Update(msg)
	case viewExplorer:
		m.explorer, cmd = m.explorer.Update(msg)
	case viewDetail:
		if m.detail != nil {
			*m.detail, cmd = m.detail.Update(msg)
		}
	case viewFilter:
		if m.filter != nil {
			*m.filter, cmd = m.filter.Update(msg)
		}
	}

	return m, cmd
}

// View renders the currently active view.
func (m AppModel) View() string {
	if m.activeView == viewFilter && m.filter != nil {
		return m.filter.View()
	}
	if m.activeView == viewDetail && m.detail != nil {
		return m.detail.View()
	}
	if m.activeView == viewExplorer {
		return m.explorer.View()
	}
	return m.feed.View()
}

// waitForHit returns a tea.Cmd that reads from the hit channel and sends a HitMsg.
func waitForHit(ch <-chan domain.Hit) tea.Cmd {
	return func() tea.Msg {
		hit, ok := <-ch
		if !ok {
			return nil
		}
		return HitMsg{Hit: hit}
	}
}

// waitForStats returns a tea.Cmd that reads from the stats channel and sends a StatsMsg.
func waitForStats(ch <-chan PollStats) tea.Cmd {
	return func() tea.Msg {
		stats, ok := <-ch
		if !ok {
			return nil
		}
		return StatsMsg{Stats: stats}
	}
}
