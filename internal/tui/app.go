package tui

import (
	"strconv"
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
	store       domain.Store
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
		store:       store,
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resizeAll(msg), nil

	case tea.KeyMsg:
		if handled, model, cmd := m.handleGlobalKey(msg); handled {
			return model, cmd
		}

	case HitMsg, EnrichmentMsg, DiscardedDomainMsg, discardTickMsg, StatsMsg:
		return m.handleStreamMsg(msg)

	case HitsLoadedMsg, DeleteHitsMsg, deleteStatusMsg, BookmarkToggleMsg:
		// Explorer-owned messages: forward to the explorer regardless of view.
		var cmd tea.Cmd
		m.explorer, cmd = m.explorer.Update(msg)
		return m, cmd

	case ShowDetailMsg:
		return m.showDetail(msg)

	case SubdomainCountMsg:
		if m.detail != nil {
			*m.detail, _ = m.detail.Update(msg)
		}
		return m, nil

	case ShowSubdomainsMsg, SwitchViewMsg, FilterAppliedMsg, FilterCancelledMsg:
		return m.handleNavMsg(msg)
	}

	return m.delegateToView(msg)
}

// handleStreamMsg processes the async streaming messages that feed the live
// view: poller hits, enrichment results, discards, ticks, and stats. Each
// re-arms its channel wait so the stream continues.
func (m AppModel) handleStreamMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HitMsg:
		return m.forwardToFeed(msg, rearm(m.hitChan, waitForHit))
	case EnrichmentMsg:
		m.applyEnrichment(msg)
		return m, rearm(m.enrichChan, waitForEnrichment)
	case DiscardedDomainMsg:
		return m.forwardToFeed(msg, rearm(m.discardChan, waitForDiscard))
	case StatsMsg:
		return m.forwardToFeed(msg, rearm(m.statsChan, waitForStats))
	case discardTickMsg:
		var cmd tea.Cmd
		m.feed, cmd = m.feed.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleNavMsg processes the view-navigation messages that switch the active
// view and (un)mount the detail/filter overlays.
func (m AppModel) handleNavMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ShowSubdomainsMsg:
		m.activeView = viewExplorer
		m.detail = nil
		return m, m.explorer.SetFilter(domain.QueryFilter{
			BaseDomain: msg.BaseDomain,
			Limit:      500,
		})

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
		return m, m.explorer.SetFilter(msg.Filter)

	case FilterCancelledMsg:
		m.activeView = viewExplorer
		m.filter = nil
		return m, nil
	}
	return m, nil
}

// showDetail opens the detail view for a hit, sizes it to the terminal, and
// kicks off the async subdomain-count query.
func (m AppModel) showDetail(msg ShowDetailMsg) (tea.Model, tea.Cmd) {
	d := NewDetailModel(msg.Hit, m.store)
	d.width = m.width
	d.height = m.height
	m.detail = &d
	m.activeView = viewDetail
	*m.detail, _ = m.detail.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return m, m.detail.Init()
}

// delegateToView forwards an otherwise-unhandled message to the active view's
// sub-model.
func (m AppModel) delegateToView(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// rearm returns the command that waits for the next value on ch, or nil when
// ch is not wired up (so the TUI never blocks on a missing channel).
func rearm[T any](ch <-chan T, wait func(<-chan T) tea.Cmd) tea.Cmd {
	if ch == nil {
		return nil
	}
	return wait(ch)
}

// resizeAll propagates a window-size change to every sub-model.
func (m AppModel) resizeAll(msg tea.WindowSizeMsg) AppModel {
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
	return m
}

// forwardToFeed delivers a message to the feed sub-model and batches the feed's
// command with an optional re-arm command (the next channel-wait), which is nil
// when the corresponding channel is not wired up.
func (m AppModel) forwardToFeed(msg tea.Msg, rearm tea.Cmd) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.feed, cmd = m.feed.Update(msg)
	if rearm == nil {
		return m, cmd
	}
	return m, tea.Batch(cmd, rearm)
}

// handleGlobalKey processes app-level key bindings (quit, tab, filter overlay)
// that take precedence over the active view. It returns handled=true when the
// key was consumed at this level; otherwise the caller delegates to the view.
func (m AppModel) handleGlobalKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	// Global quit: ctrl+c always quits, q quits unless in filter overlay.
	if msg.String() == "ctrl+c" {
		return true, m, tea.Quit
	}
	if key.Matches(msg, m.keys.Quit) && m.activeView != viewFilter {
		return true, m, tea.Quit
	}

	// Tab toggles between feed and explorer.
	if key.Matches(msg, m.keys.Tab) && m.activeView != viewFilter && m.activeView != viewDetail {
		if m.activeView == viewFeed {
			m.activeView = viewExplorer
			// Auto-reload explorer from DB when switching to it.
			m.explorer.loading = true
			return true, m, m.explorer.loadHitsCmd()
		}
		m.activeView = viewFeed
		return true, m, nil
	}

	// Filter overlay toggle.
	if key.Matches(msg, m.keys.Filter) && m.activeView == viewExplorer {
		f := NewFilterModel()
		f.width = m.width
		f.height = m.height
		m.filter = &f
		m.activeView = viewFilter
		return true, m, m.filter.Init()
	}

	return false, m, nil
}

// applyEnrichment propagates enrichment data for a domain into the feed,
// explorer cache, and the detail view if it is currently showing that domain.
func (m AppModel) applyEnrichment(msg EnrichmentMsg) {
	apply := func(h *domain.Hit) {
		h.IsLive = msg.IsLive
		h.ResolvedIPs = msg.ResolvedIPs
		h.HostingProvider = msg.HostingProvider
		h.HTTPStatus = msg.HTTPStatus
	}

	for i := range m.feed.hits {
		if m.feed.hits[i].Domain == msg.Domain {
			apply(&m.feed.hits[i])
			break
		}
	}
	for i := range m.explorer.hits {
		if m.explorer.hits[i].Domain == msg.Domain {
			apply(&m.explorer.hits[i])
			break
		}
	}
	if m.detail != nil && m.detail.hit.Domain == msg.Domain {
		apply(&m.detail.hit)
	}
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
	topBorder := border.TopLeft + border.Top + titleRendered + strings.Repeat(border.Top, remaining) + border.TopRight
	topBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(topBorder)

	// Build bottom border.
	bottomBorder := border.BottomLeft + strings.Repeat(border.Bottom, innerWidth) + border.BottomRight
	bottomBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(bottomBorder)

	// Wrap each content line with side borders.
	borderStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	leftBorder := borderStyle.Render(border.Left)
	rightBorder := borderStyle.Render(border.Right)

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
	s := strconv.FormatInt(n, 10)
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
