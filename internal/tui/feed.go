package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	maxFeedHits       = 500
	keywordSidebarMin = 100
	topKeywordsCount  = 10
)

// FeedModel displays a live scrollable feed of domain hits from CT log polling.
type FeedModel struct {
	hits        []domain.Hit
	viewport    viewport.Model
	stats       PollStats
	topKeywords []domain.KeywordCount
	profile     string
	keys        KeyMap
	width       int
	height      int
	ready       bool
}

// NewFeedModel creates a new live feed view.
func NewFeedModel(profile string) FeedModel {
	return FeedModel{
		hits:        make([]domain.Hit, 0, maxFeedHits),
		topKeywords: make([]domain.KeywordCount, 0, topKeywordsCount),
		profile:     profile,
		keys:        DefaultKeyMap(),
	}
}

// Init returns the initial command for the feed model.
func (m FeedModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the feed model.
func (m FeedModel) Update(msg tea.Msg) (FeedModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 2
		statusHeight := 1
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
	stats := fmt.Sprintf(
		" Scanned: %d | Hits: %d | Rate: %.0f certs/s | Logs: %d | Profile: %s",
		m.stats.CertsScanned,
		m.stats.HitsFound,
		m.stats.CertsPerSec,
		m.stats.ActiveLogs,
		m.profile,
	)
	return StyleStatusBar.Width(m.width).Render(stats)
}

func (m FeedModel) renderHits() string {
	if len(m.hits) == 0 {
		return StyleHelpDesc.Render("  Waiting for hits...")
	}
	var b strings.Builder
	for i, hit := range m.hits {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.renderHitLine(hit))
	}
	return b.String()
}

func (m FeedModel) renderHitLine(hit domain.Hit) string {
	ts := hit.CreatedAt.Format("15:04:05")
	sevTag := renderSeverityTag(string(hit.Severity))
	score := fmt.Sprintf("%2d", hit.Score)
	kw := strings.Join(hit.Keywords, ",")
	if len(kw) > 30 {
		kw = kw[:27] + "..."
	}
	issuer := hit.IssuerCN
	if len(issuer) > 20 {
		issuer = issuer[:17] + "..."
	}
	return fmt.Sprintf(" %s %s %s %-40s %-30s %s", ts, sevTag, score, hit.Domain, kw, issuer)
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
