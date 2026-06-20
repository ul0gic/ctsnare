package tui

import (
	"fmt"
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
	viewNetwork  = 2
	viewDetail   = 3
	viewFilter   = 4
	viewHelp     = 5
)

// AppModel is the root Bubble Tea model that manages view switching and message routing.
type AppModel struct {
	activeView  int
	prevView    int // view to restore when the help overlay is dismissed
	feed        FeedModel
	explorer    ExplorerModel
	network     NetworkModel
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

// NewApp creates a new root TUI application model. A nil store or nil channels
// are tolerated: the TUI then opens without polling or enrichment.
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
		network:     NewNetworkModel(store),
		keys:        DefaultKeyMap(),
		store:       store,
		hitChan:     hitChan,
		statsChan:   statsChan,
		enrichChan:  enrichChan,
		discardChan: discardChan,
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.explorer.Init(),
		m.network.Init(),
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

	case HitsLoadedMsg, DeleteHitsMsg, deleteStatusMsg, BookmarkToggleMsg, ClustersLoadedMsg:
		return m.handleDataMsg(msg)

	case ShowDetailMsg:
		return m.showDetail(msg)

	case SubdomainCountMsg:
		if m.detail != nil {
			*m.detail, _ = m.detail.Update(msg)
		}
		return m, nil

	case ShowSubdomainsMsg, ShowClusterMsg, SwitchViewMsg, FilterAppliedMsg, FilterCancelledMsg:
		return m.handleNavMsg(msg)
	}

	return m.delegateToView(msg)
}

// handleDataMsg routes a store result to its owning view (network owns
// ClustersLoadedMsg, explorer the rest) so off-screen tables stay current.
func (m AppModel) handleDataMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if _, ok := msg.(ClustersLoadedMsg); ok {
		m.network, cmd = m.network.Update(msg)
		return m, cmd
	}
	m.explorer, cmd = m.explorer.Update(msg)
	return m, cmd
}

// handleStreamMsg processes async feed-streaming messages (hits, enrichment,
// discards, ticks, stats), re-arming each channel wait to continue the stream.
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

	case ShowClusterMsg:
		m.activeView = viewExplorer
		m.detail = nil
		return m, m.explorer.SetFilter(domain.QueryFilter{
			SharedIP: msg.IP,
			Limit:    500,
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
	case viewNetwork:
		m.network, cmd = m.network.Update(msg)
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
	m.network, _ = m.network.Update(msg)
	if m.detail != nil {
		*m.detail, _ = m.detail.Update(msg)
	}
	if m.filter != nil {
		*m.filter, _ = m.filter.Update(msg)
	}
	return m
}

// forwardToFeed delivers a message to the feed sub-model, batching its command
// with the next channel-wait (rearm is nil when that channel isn't wired up).
func (m AppModel) forwardToFeed(msg tea.Msg, rearm tea.Cmd) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.feed, cmd = m.feed.Update(msg)
	if rearm == nil {
		return m, cmd
	}
	return m, tea.Batch(cmd, rearm)
}

// handleGlobalKey processes app-level bindings that take precedence over the
// active view, returning handled=true when it consumed the key.
func (m AppModel) handleGlobalKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return true, m, tea.Quit
	}

	// The help overlay is modal: it handles its own dismissal and swallows the rest.
	if m.activeView == viewHelp {
		return m.handleHelpKey(msg)
	}

	// q is a typeable character in the filter overlay, so don't quit there.
	if key.Matches(msg, m.keys.Quit) && m.activeView != viewFilter {
		return true, m, tea.Quit
	}

	if handled, model, cmd := m.handleOverlayKey(msg); handled {
		return true, model, cmd
	}

	if key.Matches(msg, m.keys.Tab) && m.activeView != viewFilter && m.activeView != viewDetail {
		return m.handleTab()
	}

	return false, m, nil
}

// handleOverlayKey opens the help or filter overlays. It returns handled=false
// when no overlay-opening key matched.
func (m AppModel) handleOverlayKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	// "?" is a typeable character in the filter overlay, so don't toggle help there.
	if key.Matches(msg, m.keys.Help) && m.activeView != viewFilter {
		m.prevView = m.activeView
		m.activeView = viewHelp
		return true, m, nil
	}

	if m.activeView == viewExplorer &&
		(key.Matches(msg, m.keys.Search) || key.Matches(msg, m.keys.Filter)) {
		opened := m.openFilterOverlay()
		return true, opened, opened.filter.Init()
	}

	return false, m, nil
}

// handleHelpKey processes a key while the modal help overlay is active: ?, esc,
// and q return to the previous view; all other keys are swallowed.
func (m AppModel) handleHelpKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Quit) {
		m.activeView = m.prevView
	}
	return true, m, nil
}

// openFilterOverlay mounts a fresh filter overlay sized to the terminal.
func (m AppModel) openFilterOverlay() AppModel {
	f := NewFilterModel()
	f.width = m.width
	f.height = m.height
	m.filter = &f
	m.activeView = viewFilter
	return m
}

// handleTab cycles feed → explorer → network → feed; each data view reloads
// from the DB on entry to reflect the latest committed state.
func (m AppModel) handleTab() (handled bool, model tea.Model, cmd tea.Cmd) {
	switch m.activeView {
	case viewFeed:
		m.activeView = viewExplorer
		// Fresh load reflects committed deletions, so drop the per-cycle guard.
		m.explorer.loading = true
		m.explorer.deletedSet = make(map[string]bool)
		return true, m, m.explorer.loadHitsCmd()
	case viewExplorer:
		m.activeView = viewNetwork
		var c tea.Cmd
		m.network, c = m.network.reload()
		return true, m, c
	default:
		m.activeView = viewFeed
		return true, m, nil
	}
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

func (m AppModel) View() string {
	if m.activeView == viewHelp {
		return renderHelpOverlay(m.keys, m.width, m.height)
	}
	if m.activeView == viewFilter && m.filter != nil {
		return m.filter.View()
	}
	if m.activeView == viewDetail && m.detail != nil {
		return m.detail.View()
	}
	if m.activeView == viewExplorer {
		return m.explorer.View()
	}
	if m.activeView == viewNetwork {
		return m.network.View()
	}
	return m.feed.View()
}

// renderHelpOverlay renders a centered modal of all bindings — the only
// discoverable surface for the explorer keys that don't fit the compact help bar.
func renderHelpOverlay(keys KeyMap, width, height int) string {
	title := StyleTitle.Render("Keyboard Shortcuts")

	var b strings.Builder
	for _, group := range keys.FullHelp() {
		for _, binding := range group {
			h := binding.Help()
			if h.Key == "" {
				continue
			}
			b.WriteString(StyleHelpKey.Render(fmt.Sprintf("%-10s", h.Key)))
			b.WriteString(StyleHelpDesc.Render(h.Desc))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	footer := StyleHelpKey.Render("?/esc/q") + StyleHelpDesc.Render(" close")
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", strings.TrimRight(b.String(), "\n"), "", footer)

	panelWidth := 40
	if width > 0 && width < panelWidth+4 {
		panelWidth = width - 4
	}

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		StyleBorder.Width(panelWidth).Padding(1, 2).Render(content),
	)
}

// renderTabBar renders the shared tab bar in a rounded border box, with extra
// as right-aligned metadata (e.g. hit count, time).
func renderTabBar(activeView, width int, extra string) string {
	appName := StyleAppName.Render("ctsnare")

	tabs := []struct {
		label string
		view  int
	}{
		{"Feed", viewFeed},
		{"Explorer", viewExplorer},
		{"Network", viewNetwork},
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

	titleRendered := " " + title + " "
	titleLen := lipgloss.Width(titleRendered)
	remaining := innerWidth - 1 - titleLen // 1 for the dash after corner
	if remaining < 0 {
		remaining = 0
	}
	topBorder := border.TopLeft + border.Top + titleRendered + strings.Repeat(border.Top, remaining) + border.TopRight
	topBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(topBorder)

	bottomBorder := border.BottomLeft + strings.Repeat(border.Bottom, innerWidth) + border.BottomRight
	bottomBorder = lipgloss.NewStyle().Foreground(colorSubtle).Render(bottomBorder)

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
