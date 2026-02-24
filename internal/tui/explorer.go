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
	table         table.Model
	hits          []domain.Hit
	filter        domain.QueryFilter
	sortCol       int
	sortDir       string
	loading       bool
	store         domain.Store
	keys          KeyMap
	width         int
	height        int
	ready         bool
	selected      map[int]bool
	confirmAction string // empty, "delete-single", "delete-batch", "clear-all"
	confirmDomain string // domain for single delete confirmation
}

// NewExplorerModel creates a new DB explorer view.
// The store parameter may be nil during Phase 2; it will be wired in Phase 3.
func NewExplorerModel(store domain.Store) ExplorerModel {
	cols := []table.Column{
		{Title: " ", Width: 4},
		{Title: "Severity", Width: 8},
		{Title: "Score", Width: 6},
		{Title: "Domain", Width: 38},
		{Title: "Keywords", Width: 23},
		{Title: "Issuer", Width: 18},
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
		table:    t,
		hits:     make([]domain.Hit, 0),
		selected: make(map[int]bool),
		sortCol:  1,
		sortDir:  "DESC",
		store:    store,
		keys:     DefaultKeyMap(),
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
		m.selected = make(map[int]bool)
		m.table.SetRows(m.hitsToRows())
		return m, nil

	case DeleteHitsMsg:
		// Hits were deleted — reload.
		m.selected = make(map[int]bool)
		m.loading = true
		return m, m.loadHitsCmd()

	case BookmarkToggleMsg:
		// Update the local hit's bookmark state and refresh the row.
		for i := range m.hits {
			if m.hits[i].Domain == msg.Domain {
				m.hits[i].Bookmarked = msg.Bookmarked
				break
			}
		}
		m.table.SetRows(m.hitsToRows())
		return m, nil

	case tea.KeyMsg:
		// Handle confirmation overlay first.
		if m.confirmAction != "" {
			return m.handleConfirm(msg)
		}

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

		case msg.String() == " ": // space — toggle select
			row := m.table.Cursor()
			if row >= 0 && row < len(m.hits) {
				if m.selected[row] {
					delete(m.selected, row)
				} else {
					m.selected[row] = true
				}
				m.table.SetRows(m.hitsToRows())
				// Move cursor down one row.
				m.table, cmd = m.table.Update(tea.KeyMsg{Type: tea.KeyDown})
			}
			return m, cmd

		case msg.String() == "a": // select all visible
			for i := range m.hits {
				m.selected[i] = true
			}
			m.table.SetRows(m.hitsToRows())
			return m, nil

		case msg.String() == "A": // deselect all
			m.selected = make(map[int]bool)
			m.table.SetRows(m.hitsToRows())
			return m, nil

		case msg.String() == "d": // delete single
			row := m.table.Cursor()
			if row >= 0 && row < len(m.hits) {
				m.confirmAction = "delete-single"
				m.confirmDomain = m.hits[row].Domain
			}
			return m, nil

		case msg.String() == "D": // delete selected batch
			if len(m.selected) > 0 {
				m.confirmAction = "delete-batch"
			}
			return m, nil

		case msg.String() == "C": // clear all
			m.confirmAction = "clear-all"
			return m, nil

		case msg.String() == "b": // bookmark toggle
			row := m.table.Cursor()
			if row >= 0 && row < len(m.hits) {
				return m, m.bookmarkToggleCmd(row)
			}
			return m, nil
		}

		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// handleConfirm processes key input during the confirmation overlay.
func (m ExplorerModel) handleConfirm(msg tea.KeyMsg) (ExplorerModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := m.confirmAction
		m.confirmAction = ""
		m.confirmDomain = ""

		switch action {
		case "delete-single":
			return m, m.deleteSingleCmd(m.confirmDomain)
		case "delete-batch":
			return m, m.deleteBatchCmd()
		case "clear-all":
			return m, m.clearAllCmd()
		}

	case "n", "N", "esc":
		m.confirmAction = ""
		m.confirmDomain = ""
	}

	return m, nil
}

// View renders the explorer model as a string.
func (m ExplorerModel) View() string {
	if !m.ready {
		return "Initializing explorer..."
	}

	filterBar := m.renderFilterBar()
	tableView := m.table.View()
	helpBar := m.renderHelpBar()

	result := lipgloss.JoinVertical(
		lipgloss.Left,
		filterBar,
		tableView,
		helpBar,
	)

	// Overlay confirmation prompt at the bottom if active.
	if m.confirmAction != "" {
		result += "\n" + m.renderConfirmPrompt()
	}

	return result
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

func (m ExplorerModel) renderConfirmPrompt() string {
	var prompt string
	switch m.confirmAction {
	case "delete-single":
		prompt = fmt.Sprintf("Delete hit for %s? (y/n)", m.confirmDomain)
	case "delete-batch":
		prompt = fmt.Sprintf("Delete %d selected hits? (y/n)", len(m.selected))
	case "clear-all":
		prompt = "Clear ALL hits? This cannot be undone. (y/n)"
	}
	return StyleStatusBar.Width(m.width).
		Background(colorHighSeverity).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render(" " + prompt)
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
	if m.filter.Bookmarked {
		parts = append(parts, "bookmarked:yes")
	}

	sortLabel := fmt.Sprintf("sort:%s %s", explorerColumns[m.sortCol], m.sortDir)
	hitCount := fmt.Sprintf("%d hits", len(m.hits))
	selCount := ""
	if len(m.selected) > 0 {
		selCount = fmt.Sprintf(" | %d selected", len(m.selected))
	}

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
	right := StyleHelpDesc.Render(fmt.Sprintf("%s%s | %s ", hitCount, selCount, sortLabel))
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return StyleHeader.Width(m.width).Render(left + gap + right)
}

func (m ExplorerModel) renderHelpBar() string {
	help := StyleHelpKey.Render("s") + StyleHelpDesc.Render(" sort") + "  " +
		StyleHelpKey.Render("f") + StyleHelpDesc.Render(" filter") + "  " +
		StyleHelpKey.Render("/") + StyleHelpDesc.Render(" search") + "  " +
		StyleHelpKey.Render("enter") + StyleHelpDesc.Render(" detail") + "  " +
		StyleHelpKey.Render("r") + StyleHelpDesc.Render(" refresh") + "  " +
		StyleHelpKey.Render("space") + StyleHelpDesc.Render(" select") + "  " +
		StyleHelpKey.Render("d/D") + StyleHelpDesc.Render(" delete") + "  " +
		StyleHelpKey.Render("tab") + StyleHelpDesc.Render(" switch view")
	return StyleStatusBar.Width(m.width).Render(help)
}

func (m ExplorerModel) hitsToRows() []table.Row {
	rows := make([]table.Row, 0, len(m.hits))
	for i, hit := range m.hits {
		// Checkbox column.
		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = StyleSelectedCheckbox.Render("[x]")
		}

		kw := strings.Join(hit.Keywords, ", ")
		if len(kw) > 23 {
			kw = kw[:20] + "..."
		}
		dom := hit.Domain
		if len(dom) > 34 {
			dom = dom[:31] + "..."
		}
		// Bookmark star prefix.
		if hit.Bookmarked {
			dom = StyleBookmarked.Render("*") + " " + dom
		}
		// Live domain indicator.
		if hit.IsLive {
			dom = StyleLiveDomain.Render(dom) + " " + StyleLiveDomain.Render("[L]")
		}
		issuer := hit.IssuerCN
		if len(issuer) > 18 {
			issuer = issuer[:15] + "..."
		}
		ts := hit.CreatedAt.Format("2006-01-02 15:04:05")

		// Apply severity colors.
		sevStyle := SeverityStyle(string(hit.Severity))
		sevCell := sevStyle.Render(string(hit.Severity))
		scoreCell := sevStyle.Render(fmt.Sprintf("%d", hit.Score))

		rows = append(rows, table.Row{
			checkbox,
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

func (m ExplorerModel) deleteSingleCmd(domainName string) tea.Cmd {
	if m.store == nil {
		return nil
	}
	store := m.store
	return func() tea.Msg {
		if err := store.DeleteHit(context.Background(), domainName); err != nil {
			return nil
		}
		return DeleteHitsMsg{Domains: []string{domainName}}
	}
}

func (m ExplorerModel) deleteBatchCmd() tea.Cmd {
	if m.store == nil {
		return nil
	}
	domains := make([]string, 0, len(m.selected))
	for idx := range m.selected {
		if idx < len(m.hits) {
			domains = append(domains, m.hits[idx].Domain)
		}
	}
	if len(domains) == 0 {
		return nil
	}
	store := m.store
	return func() tea.Msg {
		if err := store.DeleteHits(context.Background(), domains); err != nil {
			return nil
		}
		return DeleteHitsMsg{Domains: domains}
	}
}

func (m ExplorerModel) bookmarkToggleCmd(rowIdx int) tea.Cmd {
	if m.store == nil || rowIdx >= len(m.hits) {
		return nil
	}
	hit := m.hits[rowIdx]
	newState := !hit.Bookmarked
	store := m.store
	domainName := hit.Domain
	return func() tea.Msg {
		if err := store.SetBookmark(context.Background(), domainName, newState); err != nil {
			return nil
		}
		return BookmarkToggleMsg{Domain: domainName, Bookmarked: newState}
	}
}

func (m ExplorerModel) clearAllCmd() tea.Cmd {
	if m.store == nil {
		return nil
	}
	store := m.store
	return func() tea.Msg {
		if err := store.ClearAll(context.Background()); err != nil {
			return nil
		}
		return DeleteHitsMsg{Domains: nil}
	}
}
