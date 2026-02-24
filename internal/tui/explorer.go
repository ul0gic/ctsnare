package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

var explorerColumns = []string{
	"Severity", "Score", "Domain", "Keywords", "Issuer", "Session", "Timestamp",
}

// sortColumns maps column index to the database sort field name.
var sortColumns = []string{
	"severity", "score", "domain", "keywords", "issuer", "session", "created_at",
}

// ExplorerModel displays a filterable, sortable table of stored hits.
type ExplorerModel struct {
	table   table.Model
	hits    []domain.Hit
	filter  domain.QueryFilter
	sortCol int
	sortDir string
	loading bool
	store   domain.Store
	keys    KeyMap
	width   int
	height  int
	ready   bool
}

// NewExplorerModel creates a new DB explorer view.
// The store parameter may be nil during Phase 2; it will be wired in Phase 3.
func NewExplorerModel(store domain.Store) ExplorerModel {
	cols := []table.Column{
		{Title: "Severity", Width: 8},
		{Title: "Score", Width: 6},
		{Title: "Domain", Width: 40},
		{Title: "Keywords", Width: 25},
		{Title: "Issuer", Width: 20},
		{Title: "Session", Width: 12},
		{Title: "Timestamp", Width: 19},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorSubtle).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(colorStatusBg).
		Background(colorText).
		Bold(true)
	t.SetStyles(s)

	return ExplorerModel{
		table:   t,
		hits:    make([]domain.Hit, 0),
		sortCol: 1,
		sortDir: "DESC",
		store:   store,
		keys:    DefaultKeyMap(),
		filter: domain.QueryFilter{
			Limit:   50,
			SortBy:  "score",
			SortDir: "DESC",
		},
	}
}

// Init returns the initial command for the explorer model.
func (m ExplorerModel) Init() tea.Cmd {
	return m.loadHitsCmd()
}

// Update handles messages for the explorer model.
func (m ExplorerModel) Update(msg tea.Msg) (ExplorerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		tableHeight := m.height - 4
		if tableHeight < 3 {
			tableHeight = 3
		}
		m.table.SetWidth(m.width)
		m.table.SetHeight(tableHeight)
		m.ready = true
		return m, nil

	case HitsLoadedMsg:
		m.hits = msg.Hits
		m.loading = false
		m.table.SetRows(m.hitsToRows())
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "s":
			m.sortCol = (m.sortCol + 1) % len(sortColumns)
			m.filter.SortBy = sortColumns[m.sortCol]
			if m.sortDir == "DESC" {
				m.sortDir = "ASC"
			} else {
				m.sortDir = "DESC"
			}
			m.filter.SortDir = m.sortDir
			m.loading = true
			return m, m.loadHitsCmd()

		case msg.String() == "enter":
			row := m.table.Cursor()
			if row >= 0 && row < len(m.hits) {
				return m, func() tea.Msg {
					return ShowDetailMsg{Hit: m.hits[row]}
				}
			}

		case msg.String() == "r":
			m.loading = true
			return m, m.loadHitsCmd()
		}

		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the explorer model as a string.
func (m ExplorerModel) View() string {
	if !m.ready {
		return "Initializing explorer..."
	}

	filterBar := m.renderFilterBar()
	tableView := m.table.View()
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		filterBar,
		tableView,
		helpBar,
	)
}

// SetFilter updates the active query filter and triggers a reload.
func (m *ExplorerModel) SetFilter(f domain.QueryFilter) tea.Cmd {
	f.SortBy = m.filter.SortBy
	f.SortDir = m.filter.SortDir
	if f.Limit == 0 {
		f.Limit = 50
	}
	m.filter = f
	m.loading = true
	return m.loadHitsCmd()
}

func (m ExplorerModel) renderFilterBar() string {
	var parts []string
	if m.filter.Keyword != "" {
		parts = append(parts, fmt.Sprintf("keyword:%s", m.filter.Keyword))
	}
	if m.filter.ScoreMin > 0 {
		parts = append(parts, fmt.Sprintf("score>=%d", m.filter.ScoreMin))
	}
	if m.filter.Severity != "" {
		parts = append(parts, fmt.Sprintf("severity:%s", m.filter.Severity))
	}
	if m.filter.Session != "" {
		parts = append(parts, fmt.Sprintf("session:%s", m.filter.Session))
	}

	sortLabel := fmt.Sprintf("sort:%s %s", explorerColumns[m.sortCol], m.sortDir)
	hitCount := fmt.Sprintf("%d hits", len(m.hits))

	var filterStr string
	if len(parts) > 0 {
		filterStr = strings.Join(parts, " | ")
	} else {
		filterStr = "no filters"
	}

	if m.loading {
		filterStr = "loading..."
	}

	left := StyleHelpDesc.Render(fmt.Sprintf(" Filters: %s", filterStr))
	right := StyleHelpDesc.Render(fmt.Sprintf("%s | %s ", hitCount, sortLabel))
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return StyleHeader.Width(m.width).Render(left + gap + right)
}

func (m ExplorerModel) renderHelpBar() string {
	help := StyleHelpKey.Render("s") + StyleHelpDesc.Render(" sort") + "  " +
		StyleHelpKey.Render("f") + StyleHelpDesc.Render(" filter") + "  " +
		StyleHelpKey.Render("/") + StyleHelpDesc.Render(" search") + "  " +
		StyleHelpKey.Render("enter") + StyleHelpDesc.Render(" detail") + "  " +
		StyleHelpKey.Render("r") + StyleHelpDesc.Render(" refresh") + "  " +
		StyleHelpKey.Render("tab") + StyleHelpDesc.Render(" switch view")
	return StyleStatusBar.Width(m.width).Render(help)
}

func (m ExplorerModel) hitsToRows() []table.Row {
	rows := make([]table.Row, 0, len(m.hits))
	for _, hit := range m.hits {
		kw := strings.Join(hit.Keywords, ", ")
		if len(kw) > 25 {
			kw = kw[:22] + "..."
		}
		dom := hit.Domain
		if len(dom) > 40 {
			dom = dom[:37] + "..."
		}
		issuer := hit.IssuerCN
		if len(issuer) > 20 {
			issuer = issuer[:17] + "..."
		}
		ts := hit.CreatedAt.Format("2006-01-02 15:04:05")

		// Apply severity colors to the severity and score columns.
		sevStyle := SeverityStyle(string(hit.Severity))
		sevCell := sevStyle.Render(string(hit.Severity))
		scoreCell := sevStyle.Render(fmt.Sprintf("%d", hit.Score))

		rows = append(rows, table.Row{
			sevCell,
			scoreCell,
			dom,
			kw,
			issuer,
			hit.Session,
			ts,
		})
	}
	return rows
}

func (m ExplorerModel) loadHitsCmd() tea.Cmd {
	if m.store == nil {
		return func() tea.Msg {
			return HitsLoadedMsg{Hits: nil}
		}
	}
	filter := m.filter
	store := m.store
	return func() tea.Msg {
		hits, err := store.QueryHits(context.Background(), filter)
		if err != nil {
			return HitsLoadedMsg{Hits: nil}
		}
		return HitsLoadedMsg{Hits: hits}
	}
}
