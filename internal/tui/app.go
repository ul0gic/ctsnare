package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/enrichment"
)

const (
	viewFeed     = 0
	viewExplorer = 1
	viewDetail   = 2
	viewFilter   = 3
)

// AppModel is the root Bubble Tea model that manages view switching and message routing.
type AppModel struct {
	activeView  int
	feed        FeedModel
	explorer    ExplorerModel
	detail      *DetailModel
	filter      *FilterModel
	keys        KeyMap
	width       int
	height      int
	hitChan     <-chan domain.Hit
	statsChan   <-chan PollStats
	enrichChan  <-chan enrichment.EnrichResult
	discardChan <-chan string
}

// NewApp creates a new root TUI application model.
// The store may be nil during Phase 2; real wiring happens in Phase 3.
// Channels may be nil if the TUI is opened without polling or enrichment.
func NewApp(
	store domain.Store,
	hitChan <-chan domain.Hit,
	statsChan <-chan PollStats,
	enrichChan <-chan enrichment.EnrichResult,
	discardChan <-chan string,
	profile string,
) AppModel {
	return AppModel{
		activeView:  viewFeed,
		feed:        NewFeedModel(profile),
		explorer:    NewExplorerModel(store),
		keys:        DefaultKeyMap(),
		hitChan:     hitChan,
		statsChan:   statsChan,
		enrichChan:  enrichChan,
		discardChan: discardChan,
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
	if m.enrichChan != nil {
		cmds = append(cmds, waitForEnrichment(m.enrichChan))
	}
	if m.discardChan != nil {
		cmds = append(cmds, waitForDiscard(m.discardChan))
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
			var cmd tea.Cmd
			if m.activeView == viewFeed {
				m.activeView = viewExplorer
				// Auto-reload explorer from DB when switching to it.
				m.explorer.loading = true
				cmd = m.explorer.loadHitsCmd()
			} else {
				m.activeView = viewFeed
			}
			return m, cmd
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

	case EnrichmentMsg:
		// Update the matching hit in the feed with enrichment data.
		for i := range m.feed.hits {
			if m.feed.hits[i].Domain == msg.Domain {
				m.feed.hits[i].IsLive = msg.IsLive
				m.feed.hits[i].ResolvedIPs = msg.ResolvedIPs
				m.feed.hits[i].HostingProvider = msg.HostingProvider
				m.feed.hits[i].HTTPStatus = msg.HTTPStatus
				break
			}
		}
		// Also update the explorer's cached hit if it matches.
		for i := range m.explorer.hits {
			if m.explorer.hits[i].Domain == msg.Domain {
				m.explorer.hits[i].IsLive = msg.IsLive
				m.explorer.hits[i].ResolvedIPs = msg.ResolvedIPs
				m.explorer.hits[i].HostingProvider = msg.HostingProvider
				m.explorer.hits[i].HTTPStatus = msg.HTTPStatus
				break
			}
		}
		// Refresh the detail view if showing this hit.
		if m.detail != nil && m.detail.hit.Domain == msg.Domain {
			m.detail.hit.IsLive = msg.IsLive
			m.detail.hit.ResolvedIPs = msg.ResolvedIPs
			m.detail.hit.HostingProvider = msg.HostingProvider
			m.detail.hit.HTTPStatus = msg.HTTPStatus
		}
		if m.enrichChan != nil {
			cmds = append(cmds, waitForEnrichment(m.enrichChan))
		}
		return m, tea.Batch(cmds...)

	case DiscardedDomainMsg:
		var cmd tea.Cmd
		m.feed, cmd = m.feed.Update(msg)
		cmds = append(cmds, cmd)
		if m.discardChan != nil {
			cmds = append(cmds, waitForDiscard(m.discardChan))
		}
		return m, tea.Batch(cmds...)

	case discardTickMsg:
		var cmd tea.Cmd
		m.feed, cmd = m.feed.Update(msg)
		return m, cmd

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

	case DeleteHitsMsg:
		var cmd tea.Cmd
		m.explorer, cmd = m.explorer.Update(msg)
		return m, cmd

	case deleteStatusMsg:
		var cmd tea.Cmd
		m.explorer, cmd = m.explorer.Update(msg)
		return m, cmd

	case BookmarkToggleMsg:
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

// --- Shared rendering helpers for Option B layout ---

// renderTabBar renders the shared tab bar wrapped in a rounded border box.
// activeView is one of viewFeed, viewExplorer, viewDetail.
// extra is right-aligned metadata (e.g. hit count, time).
func renderTabBar(activeView, width int, extra string) string {
	appName := StyleAppName.Render("ctsnare")

	tabs := []struct {
		label string
		view  int
	}{
		{"Feed", viewFeed},
		{"Explorer", viewExplorer},
	}
	if activeView == viewDetail {
		tabs = append(tabs, struct {
			label string
			view  int
		}{"Detail", viewDetail})
	}

	var tabParts []string
	for _, t := range tabs {
		if t.view == activeView {
			tabParts = append(tabParts, StyleTabActive.Render(t.label))
		} else {
			tabParts = append(tabParts, StyleTabInactive.Render(t.label))
		}
	}

	left := " " + appName + "  " + strings.Join(tabParts, " ")
	right := ""
	if extra != "" {
		right = StyleHelpDesc.Render(extra) + " "
	}

	innerWidth := width - 2 // account for left+right border chars
	if innerWidth < 1 {
		innerWidth = 1
	}
	gap := strings.Repeat(" ", max(0, innerWidth-lipgloss.Width(left)-lipgloss.Width(right)))
	content := left + gap + right

	return StylePanel.Width(width - 2).Render(content)
}

// renderTitledPanel wraps content in a rounded border box with a title inlined in the top border.
// The title appears after the top-left corner: ╭─ Title ───...─╮
func renderTitledPanel(title, content string, width int) string {
	border := lipgloss.RoundedBorder()
	innerWidth := width - 2 // left + right border chars
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Build the custom top border with the title embedded.
	titleRendered := " " + title + " "
	titleLen := lipgloss.Width(titleRendered)
	remaining := innerWidth - 1 - titleLen // 1 for the dash after corner
	if remaining < 0 {
		remaining = 0
	}
	topBorder := string(border.TopLeft) + string(border.Top) + titleRendered + strings.Repeat(string(border.Top), remaining) + string(border.TopRight)
	topBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(topBorder)

	// Build bottom border.
	bottomBorder := string(border.BottomLeft) + strings.Repeat(string(border.Bottom), innerWidth) + string(border.BottomRight)
	bottomBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(bottomBorder)

	// Wrap each content line with side borders.
	borderStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	leftBorder := borderStyle.Render(string(border.Left))
	rightBorder := borderStyle.Render(string(border.Right))

	lines := strings.Split(content, "\n")
	var body strings.Builder
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		pad := strings.Repeat(" ", max(0, innerWidth-lineWidth))
		body.WriteString(leftBorder + line + pad + rightBorder + "\n")
	}

	return topBorder + "\n" + body.String() + bottomBorder
}

// formatClock returns the current time formatted as HH:MM.
func formatClock() string {
	return time.Now().Format("15:04")
}

// formatNumber adds commas to a number for readability (e.g. 12847 -> "12,847").
func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
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

// waitForEnrichment returns a tea.Cmd that reads from the enrichment channel
// and converts the result to an EnrichmentMsg for TUI consumption.
func waitForEnrichment(ch <-chan enrichment.EnrichResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-ch
		if !ok {
			return nil
		}
		return EnrichmentMsg{
			Domain:          result.Domain,
			IsLive:          result.IsLive,
			ResolvedIPs:     result.ResolvedIPs,
			HostingProvider: result.HostingProvider,
			HTTPStatus:      result.HTTPStatus,
		}
	}
}

// waitForDiscard returns a tea.Cmd that reads from the discard channel
// and converts the domain name to a DiscardedDomainMsg.
func waitForDiscard(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		domain, ok := <-ch
		if !ok {
			return nil
		}
		return DiscardedDomainMsg{Domain: domain}
	}
}
