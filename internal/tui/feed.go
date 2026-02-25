package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	maxFeedHits       = 500
	keywordSidebarMin = 120 // sidebar panel appears at 120+ cols
	topKeywordsCount  = 10
	sidebarPanelWidth = 26 // including panel borders
)

// discardTickMsg fires periodically to keep the TUI responsive.
type discardTickMsg time.Time

// FeedModel displays a live scrollable feed of domain hits from CT log polling.
type FeedModel struct {
	hits         []domain.Hit
	discardCount int64
	lowCount     int64 // LOW-scored hits filtered from display
	viewport     viewport.Model
	stats        PollStats
	topKeywords  []domain.KeywordCount
	profile      string
	keys         KeyMap
	width        int
	height       int
	ready        bool
	autoScroll   bool // true when viewport tracks newest entries (top)
	paused       bool // when true, new hits are counted but not added to display
}

// NewFeedModel creates a new live feed view.
func NewFeedModel(profile string) FeedModel {
	return FeedModel{
		hits:        make([]domain.Hit, 0, maxFeedHits),
		topKeywords: make([]domain.KeywordCount, 0, topKeywordsCount),
		profile:     profile,
		keys:        DefaultKeyMap(),
		autoScroll:  true,
	}
}

// Init returns the initial command for the feed model.
func (m FeedModel) Init() tea.Cmd {
	return tickDiscards()
}

// Update handles messages for the feed model.
func (m FeedModel) Update(msg tea.Msg) (FeedModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Layout: tabBar(3) + feedPanel(top+bottom border = 2) + statsPanel(3) + helpBar(1) = 9 lines of chrome
		contentHeight := m.height - 9
		if contentHeight < 1 {
			contentHeight = 1
		}
		contentWidth := m.feedContentWidth()
		if !m.ready {
			m.viewport = viewport.New(contentWidth, contentHeight)
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = contentHeight
		}
		m.viewport.SetContent(m.renderHits())
		return m, nil

	case HitMsg:
		// Filter LOW-scored hits from the feed — they're heuristic-only noise.
		if msg.Hit.Score < 4 {
			m.lowCount++
			return m, nil
		}
		m.topKeywords = updateKeywordCounts(m.topKeywords, msg.Hit.Keywords)
		if m.paused {
			return m, nil
		}
		m.hits = prependHit(m.hits, msg.Hit, maxFeedHits)
		if m.ready {
			m.viewport.SetContent(m.renderHits())
			if m.autoScroll {
				m.viewport.GotoTop()
			}
		}
		return m, nil

	case DiscardedDomainMsg:
		m.discardCount++
		return m, nil

	case discardTickMsg:
		// Tick still fires to keep the TUI responsive; just re-subscribe.
		return m, tickDiscards()

	case StatsMsg:
		m.stats = msg.Stats
		return m, nil

	case tea.KeyMsg:
		// Toggle pause with 'p'.
		if key.Matches(msg, m.keys.Pause) {
			m.paused = !m.paused
			return m, nil
		}
		if m.ready {
			m.viewport, cmd = m.viewport.Update(msg)
			// If user scrolled away from top, pause auto-scroll.
			// Re-enable when they return to the top.
			m.autoScroll = m.viewport.YOffset == 0
		}
		return m, cmd
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		m.autoScroll = m.viewport.YOffset == 0
	}
	return m, cmd
}

// View renders the feed model as a string.
func (m FeedModel) View() string {
	if !m.ready {
		return "Initializing feed..."
	}

	// Tab bar with live/paused indicator and profile.
	var liveTag string
	if m.paused {
		liveTag = lipgloss.NewStyle().Foreground(colorHighSeverity).Bold(true).Render("PAUSED")
	} else if !m.autoScroll {
		liveTag = lipgloss.NewStyle().Foreground(colorMedSeverity).Bold(true).Render("SCROLL-PAUSED")
	} else {
		liveTag = StyleLiveDomain.Render("LIVE")
	}
	tabExtra := liveTag + " " + StyleHelpDesc.Render("("+m.profile+")") + " " + StyleHelpDesc.Render(formatClock())
	tabBar := renderTabBar(viewFeed, m.width, tabExtra)

	// Feed content wrapped in titled panel.
	feedPanel := m.renderFeedPanel()

	// Stats wrapped in titled panel.
	statsPanel := m.renderStatsPanel()

	// Help bar sits below all panels.
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabBar,
		feedPanel,
		statsPanel,
		helpBar,
	)
}

// feedContentWidth returns the width available for the viewport content inside the feed panel.
func (m FeedModel) feedContentWidth() int {
	// Subtract 2 for the panel's left+right border chars.
	w := m.width - 2
	if m.width >= keywordSidebarMin && len(m.topKeywords) > 0 {
		w -= sidebarPanelWidth
	}
	if w < 20 {
		w = 20
	}
	return w
}

// renderFeedPanel renders the feed viewport wrapped in a titled panel,
// with an optional sidebar panel at 120+ columns.
func (m FeedModel) renderFeedPanel() string {
	feedContent := m.viewport.View()
	hasSidebar := m.width >= keywordSidebarMin && len(m.topKeywords) > 0

	if hasSidebar {
		// Feed panel takes remaining width after sidebar.
		feedPanelWidth := m.width - sidebarPanelWidth
		feedBox := renderTitledPanel("Live Feed", feedContent, feedPanelWidth)

		// Sidebar panel.
		sidebarContent := m.renderSidebarContent()
		sidebarBox := renderTitledPanel("Top Keywords", sidebarContent, sidebarPanelWidth)

		return lipgloss.JoinHorizontal(lipgloss.Top, feedBox, sidebarBox)
	}

	return renderTitledPanel("Live Feed", feedContent, m.width)
}

// renderStatsPanel renders the stats bar wrapped in a titled panel.
func (m FeedModel) renderStatsPanel() string {
	scanned := StyleHelpDesc.Render("Scanned ") + lipgloss.NewStyle().Foreground(colorLowSeverity).Render(formatNumber(m.stats.CertsScanned))
	hits := StyleHelpDesc.Render("  Hits ") + lipgloss.NewStyle().Foreground(colorMedSeverity).Render(fmt.Sprintf("%d", m.stats.HitsFound))

	rateColor := colorLowSeverity
	if m.stats.CertsPerSec == 0 {
		rateColor = colorHighSeverity
	}
	rate := StyleHelpDesc.Render("  Rate ") + lipgloss.NewStyle().Foreground(rateColor).Render(fmt.Sprintf("%.0f c/s", m.stats.CertsPerSec))
	hpm := StyleHelpDesc.Render("  Hits/min ") + lipgloss.NewStyle().Foreground(colorMedSeverity).Render(fmt.Sprintf("%.1f", m.stats.HitsPerMin))
	logs := StyleHelpDesc.Render("  Logs ") + lipgloss.NewStyle().Foreground(colorLowSeverity).Render(fmt.Sprintf("%d", m.stats.ActiveLogs))

	statsLine := " " + scanned + hits + rate + hpm + logs

	// Add extras at wider widths.
	if m.width >= 100 {
		low := StyleHelpDesc.Render("  Low ") + StyleHelpDesc.Render(formatNumber(m.lowCount))
		discarded := StyleHelpDesc.Render("  Discarded ") + StyleHelpDesc.Render(formatNumber(m.discardCount))
		prof := StyleHelpDesc.Render("  Profile ") + StyleHelpDesc.Render(m.profile)
		statsLine += low + discarded + prof
	}

	return renderTitledPanel("Stats", statsLine, m.width)
}

func (m FeedModel) renderHelpBar() string {
	sep := StyleHelpDesc.Render("  ")
	help := StyleHelpKey.Render("Tab") + StyleHelpDesc.Render("=views") + sep +
		StyleHelpKey.Render("p") + StyleHelpDesc.Render("=pause") + sep +
		StyleHelpKey.Render("j/k") + StyleHelpDesc.Render("=scroll") + sep +
		StyleHelpKey.Render("q") + StyleHelpDesc.Render("=quit")
	return " " + help
}

func (m FeedModel) renderHits() string {
	if len(m.hits) == 0 {
		return StyleHelpDesc.Render("  Waiting for hits...")
	}

	var b strings.Builder

	// Column header row.
	b.WriteString(m.renderHeaderLine())
	b.WriteByte('\n')

	for _, hit := range m.hits {
		b.WriteByte('\n')
		b.WriteString(m.renderHitLine(hit))
	}

	return b.String()
}

func (m FeedModel) renderHeaderLine() string {
	headerStyle := StyleHelpDesc

	domWidth := 35
	if m.width >= 100 {
		domWidth = 38
	}

	line := " " + headerStyle.Render(fmt.Sprintf("%-8s", "TIME")) +
		" " + headerStyle.Render(fmt.Sprintf(" %-4s", "SEV")) +
		" " + headerStyle.Render(fmt.Sprintf("%2s", "SC")) +
		" " + headerStyle.Render(fmt.Sprintf("%-*s", domWidth, "DOMAIN"))

	if m.width >= 80 {
		line += " " + headerStyle.Render(fmt.Sprintf("%-22s", "KEYWORDS"))
	}
	if m.width >= 100 {
		line += " " + headerStyle.Render(fmt.Sprintf("%-15s", "ISSUER"))
	}

	return line
}

func (m FeedModel) renderHitLine(hit domain.Hit) string {
	ts := hit.CreatedAt.Format("15:04:05")

	sev := string(hit.Severity)
	sevStyle := SeverityStyle(sev)

	// Pad plain text to fixed widths BEFORE applying ANSI styles.
	sevTag := sevStyle.Render(fmt.Sprintf(" %-4s", sev))
	score := sevStyle.Render(fmt.Sprintf("%2d", hit.Score))

	// Responsive domain width: use available space.
	domWidth := 35
	if m.width >= 100 {
		domWidth = 38
	}

	domainStr := hit.Domain
	if len(domainStr) > domWidth {
		domainStr = domainStr[:domWidth-3] + "..."
	}
	paddedDomain := fmt.Sprintf("%-*s", domWidth, domainStr)
	domainRendered := sevStyle.Render(paddedDomain)
	if hit.IsLive {
		domainRendered = StyleLiveDomain.Render(paddedDomain)
	}

	line := " " + ts + " " + sevTag + " " + score + " " + domainRendered

	// Responsive columns: keywords at 80+, issuer at 100+.
	if m.width >= 80 {
		kw := strings.Join(hit.Keywords, ",")
		if kw == "" {
			kw = "—"
		}
		kwWidth := 22
		if len(kw) > kwWidth {
			kw = kw[:kwWidth-3] + "..."
		}
		kwRendered := fmt.Sprintf("%-*s", kwWidth, kw)
		line += " " + kwRendered
	}

	if m.width >= 100 {
		issuer := hit.IssuerCN
		issuerWidth := 15
		if len(issuer) > issuerWidth {
			issuer = issuer[:issuerWidth-3] + "..."
		}
		issuerRendered := fmt.Sprintf("%-*s", issuerWidth, issuer)
		line += " " + issuerRendered
	}

	return line
}

func renderSeverityTag(severity string) string {
	style := SeverityStyle(severity)
	return style.Render(fmt.Sprintf("[%-4s]", severity))
}

// renderSidebarContent renders the keyword list content (without panel wrapping).
func (m FeedModel) renderSidebarContent() string {
	var b strings.Builder
	for i, kw := range m.topKeywords {
		if i >= topKeywordsCount {
			break
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		keyword := kw.Keyword
		if len(keyword) > 14 {
			keyword = keyword[:11] + "..."
		}
		b.WriteString(fmt.Sprintf(" %2d. %-14s %d", i+1, keyword, kw.Count))
	}
	return b.String()
}

// tickDiscards returns a command that fires a discard tick after 500ms.
func tickDiscards() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return discardTickMsg(t)
	})
}

// prependHit adds a hit to the front of the slice, maintaining the max capacity.
func prependHit(hits []domain.Hit, hit domain.Hit, maxSize int) []domain.Hit {
	hits = append([]domain.Hit{hit}, hits...)
	if len(hits) > maxSize {
		hits = hits[:maxSize]
	}
	return hits
}

// updateKeywordCounts updates the running keyword frequency counts.
func updateKeywordCounts(counts []domain.KeywordCount, keywords []string) []domain.KeywordCount {
	freq := make(map[string]int, len(counts))
	for _, kc := range counts {
		freq[kc.Keyword] = kc.Count
	}
	for _, kw := range keywords {
		freq[kw]++
	}

	result := make([]domain.KeywordCount, 0, len(freq))
	for kw, count := range freq {
		result = append(result, domain.KeywordCount{Keyword: kw, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	if len(result) > topKeywordsCount {
		result = result[:topKeywordsCount]
	}
	return result
}
