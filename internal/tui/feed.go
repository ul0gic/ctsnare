package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	maxFeedHits       = 500
	keywordSidebarMin = 100
	topKeywordsCount  = 10
	maxDiscards       = 50
	discardFadeSecs   = 2
)

// discardEntry tracks a discarded (zero-score) domain with an expiry time.
type discardEntry struct {
	Domain string
	FadeAt time.Time
}

// discardTickMsg fires periodically to expire old discards from the feed.
type discardTickMsg time.Time

// FeedModel displays a live scrollable feed of domain hits from CT log polling.
type FeedModel struct {
	hits         []domain.Hit
	discards     []discardEntry
	discardCount int64
	viewport     viewport.Model
	stats        PollStats
	topKeywords  []domain.KeywordCount
	profile      string
	keys         KeyMap
	width        int
	height       int
	ready        bool
}

// NewFeedModel creates a new live feed view.
func NewFeedModel(profile string) FeedModel {
	return FeedModel{
		hits:        make([]domain.Hit, 0, maxFeedHits),
		discards:    make([]discardEntry, 0, maxDiscards),
		topKeywords: make([]domain.KeywordCount, 0, topKeywordsCount),
		profile:     profile,
		keys:        DefaultKeyMap(),
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
		headerHeight := 2
		statusHeight := 2 // status bar + help bar
		contentHeight := m.height - headerHeight - statusHeight
		if contentHeight < 1 {
			contentHeight = 1
		}
		contentWidth := m.contentWidth()
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
		m.hits = prependHit(m.hits, msg.Hit, maxFeedHits)
		m.topKeywords = updateKeywordCounts(m.topKeywords, msg.Hit.Keywords)
		if m.ready {
			m.viewport.SetContent(m.renderHits())
		}
		return m, nil

	case DiscardedDomainMsg:
		m.discardCount++
		entry := discardEntry{
			Domain: msg.Domain,
			FadeAt: time.Now().Add(discardFadeSecs * time.Second),
		}
		m.discards = append([]discardEntry{entry}, m.discards...)
		if len(m.discards) > maxDiscards {
			m.discards = m.discards[:maxDiscards]
		}
		if m.ready {
			m.viewport.SetContent(m.renderHits())
		}
		return m, nil

	case discardTickMsg:
		now := time.Now()
		filtered := m.discards[:0]
		for _, d := range m.discards {
			if d.FadeAt.After(now) {
				filtered = append(filtered, d)
			}
		}
		m.discards = filtered
		if m.ready {
			m.viewport.SetContent(m.renderHits())
		}
		return m, tickDiscards()

	case StatsMsg:
		m.stats = msg.Stats
		return m, nil

	case tea.KeyMsg:
		if m.ready {
			m.viewport, cmd = m.viewport.Update(msg)
		}
		return m, cmd
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

// View renders the feed model as a string.
func (m FeedModel) View() string {
	if !m.ready {
		return "Initializing feed..."
	}

	header := m.renderHeader()
	status := m.renderStatusBar()
	helpBar := m.renderHelpBar()

	mainContent := m.viewport.View()

	if m.width >= keywordSidebarMin && len(m.topKeywords) > 0 {
		sidebar := m.renderSidebar()
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			mainContent,
			sidebar,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		mainContent,
		status,
		helpBar,
	)
}

func (m FeedModel) contentWidth() int {
	if m.width >= keywordSidebarMin {
		return m.width - 26
	}
	return m.width
}

func (m FeedModel) renderHeader() string {
	title := StyleTitle.Render("Live Feed")
	profileTag := StyleHelpDesc.Render(fmt.Sprintf("[%s]", m.profile))
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)-lipgloss.Width(profileTag)))
	return StyleHeader.Width(m.width).Render(title + gap + profileTag)
}

func (m FeedModel) renderStatusBar() string {
	scanned := StyleHelpDesc.Render("Scanned: ") + lipgloss.NewStyle().Foreground(colorLowSeverity).Render(fmt.Sprintf("%d", m.stats.CertsScanned))
	hits := StyleHelpDesc.Render(" | Hits: ") + lipgloss.NewStyle().Foreground(colorMedSeverity).Render(fmt.Sprintf("%d", m.stats.HitsFound))

	rateColor := colorLowSeverity
	if m.stats.CertsPerSec == 0 {
		rateColor = colorHighSeverity
	}
	rate := StyleHelpDesc.Render(" | Rate: ") + lipgloss.NewStyle().Foreground(rateColor).Render(fmt.Sprintf("%.0f certs/s", m.stats.CertsPerSec))

	hpm := StyleHelpDesc.Render(" | Hits/min: ") + lipgloss.NewStyle().Foreground(colorMedSeverity).Render(fmt.Sprintf("%.1f", m.stats.HitsPerMin))
	discarded := StyleHelpDesc.Render(" | Discarded: ") + StyleHelpDesc.Render(fmt.Sprintf("%d", m.discardCount))
	logs := StyleHelpDesc.Render(" | Logs: ") + lipgloss.NewStyle().Foreground(colorLowSeverity).Render(fmt.Sprintf("%d", m.stats.ActiveLogs))
	prof := StyleHelpDesc.Render(" | Profile: ") + StyleHelpDesc.Render(m.profile)

	return StyleStatusBar.Width(m.width).Render(" " + scanned + hits + rate + hpm + discarded + logs + prof)
}

func (m FeedModel) renderHelpBar() string {
	help := StyleHelpKey.Render("tab") + StyleHelpDesc.Render("=views") + "  " +
		StyleHelpKey.Render("q") + StyleHelpDesc.Render("=quit") + "  " +
		StyleHelpKey.Render("?") + StyleHelpDesc.Render("=help") + "  " +
		StyleHelpKey.Render("j/k") + StyleHelpDesc.Render("=scroll")
	return StyleStatusBar.Width(m.width).Render(" " + help)
}

func (m FeedModel) renderHits() string {
	if len(m.hits) == 0 && len(m.discards) == 0 {
		return StyleHelpDesc.Render("  Waiting for hits...")
	}

	var b strings.Builder
	hitIdx := 0
	discardIdx := 0

	// Interleave hits and discards. Hits are primary; insert discards between them.
	for hitIdx < len(m.hits) || discardIdx < len(m.discards) {
		// Show a hit line.
		if hitIdx < len(m.hits) {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(m.renderHitLine(m.hits[hitIdx]))
			hitIdx++
		}

		// Interleave up to 2 discards after each hit to keep feed alive.
		for i := 0; i < 2 && discardIdx < len(m.discards); i++ {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(m.renderDiscardLine(m.discards[discardIdx]))
			discardIdx++
		}
	}

	return b.String()
}

func (m FeedModel) renderHitLine(hit domain.Hit) string {
	ts := hit.CreatedAt.Format("15:04:05")
	sev := string(hit.Severity)
	sevStyle := SeverityStyle(sev)
	sevTag := sevStyle.Render(fmt.Sprintf("[%-4s]", sev))
	score := sevStyle.Render(fmt.Sprintf("%2d", hit.Score))
	domainStr := hit.Domain
	if len(domainStr) > 40 {
		domainStr = domainStr[:37] + "..."
	}
	domainRendered := sevStyle.Render(domainStr)
	kw := strings.Join(hit.Keywords, ",")
	if len(kw) > 30 {
		kw = kw[:27] + "..."
	}
	issuer := hit.IssuerCN
	if len(issuer) > 20 {
		issuer = issuer[:17] + "..."
	}
	return fmt.Sprintf(" %s %s %s %-40s %-30s %s", ts, sevTag, score, domainRendered, kw, issuer)
}

func (m FeedModel) renderDiscardLine(d discardEntry) string {
	domainStr := d.Domain
	if len(domainStr) > 60 {
		domainStr = domainStr[:57] + "..."
	}
	return StyleDiscardedDomain.Render(fmt.Sprintf("          %-60s", domainStr))
}

func renderSeverityTag(severity string) string {
	style := SeverityStyle(severity)
	return style.Render(fmt.Sprintf("[%-4s]", severity))
}

func (m FeedModel) renderSidebar() string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render("Top Keywords"))
	b.WriteByte('\n')
	for i, kw := range m.topKeywords {
		if i >= topKeywordsCount {
			break
		}
		b.WriteString(fmt.Sprintf(" %2d. %-14s %d\n", i+1, kw.Keyword, kw.Count))
	}
	return StyleBorder.Width(24).Render(b.String())
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
