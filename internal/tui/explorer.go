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

// deleteStatusMsg provides feedback after a delete operation.
type deleteStatusMsg struct {
	Success bool
	Count   int
	Err     error
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
	keepSelection bool            // preserve selection across the next reload (e.g. sort)
	confirmAction string          // empty, "delete-single", "delete-batch", "clear-all"
	confirmDomain string          // domain for single delete confirmation
	deletedSet    map[string]bool // recently deleted domains, filtered from reloads
	statusText    string          // brief status message shown in filter bar
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
		table:      t,
		hits:       make([]domain.Hit, 0),
		selected:   make(map[int]bool),
		deletedSet: make(map[string]bool),
		sortCol:    1,
		sortDir:    "DESC",
		store:      store,
		keys:       DefaultKeyMap(),
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
		// Layout: tabBar(3) + panel top border(1) + table header+border(2) + panel bottom border(1) + helpBar(1) = 8 lines chrome
		tableHeight := m.height - 8
		if tableHeight < 3 {
			tableHeight = 3
		}
		// Table width fits inside the panel borders (2 chars for left+right).
		m.table.SetWidth(m.width - 2)
		m.table.SetHeight(tableHeight)
		m.ready = true
		return m, nil

	case HitsLoadedMsg:
		// Filter out recently deleted domains so the poller can't re-insert them visually.
		filtered := make([]domain.Hit, 0, len(msg.Hits))
		for _, h := range msg.Hits {
			if !m.deletedSet[h.Domain] {
				filtered = append(filtered, h)
			}
		}

		if m.keepSelection {
			// Remap selection by domain identity across reloads (e.g. sort change).
			oldDomains := make(map[string]bool, len(m.selected))
			for idx := range m.selected {
				if idx < len(m.hits) {
					oldDomains[m.hits[idx].Domain] = true
				}
			}
			m.hits = filtered
			m.selected = make(map[int]bool)
			for i, h := range m.hits {
				if oldDomains[h.Domain] {
					m.selected[i] = true
				}
			}
			m.keepSelection = false
		} else {
			m.hits = filtered
			m.selected = make(map[int]bool)
		}
		m.loading = false
		m.table.SetRows(m.hitsToRows())
		return m, nil

	case DeleteHitsMsg:
		// Hits were deleted -- reload.
		m.selected = make(map[int]bool)
		m.loading = true
		return m, m.loadHitsCmd()

	case deleteStatusMsg:
		if msg.Success {
			m.statusText = fmt.Sprintf("Deleted %d hit(s)", msg.Count)
		} else {
			m.statusText = fmt.Sprintf("Delete failed: %v", msg.Err)
		}
		return m, nil

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

		// Clear status message on any key press.
		m.statusText = ""

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
			m.keepSelection = true
			return m, m.loadHitsCmd()

		case msg.String() == "enter":
			row := m.table.Cursor()
			if row >= 0 && row < len(m.hits) {
				return m, func() tea.Msg {
					return ShowDetailMsg{Hit: m.hits[row]}
				}
			}

		case msg.String() == "r":
			// Explicit reload clears the deleted set and status.
			m.deletedSet = make(map[string]bool)
			m.statusText = ""
			m.loading = true
			return m, m.loadHitsCmd()

		case msg.String() == " ": // space -- toggle select
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
		domainName := m.confirmDomain
		m.confirmAction = ""
		m.confirmDomain = ""

		switch action {
		case "delete-single":
			// Track deleted domain so poller re-inserts don't bring it back.
			m.deletedSet[domainName] = true
			return m, m.deleteSingleCmd(domainName)
		case "delete-batch":
			for idx := range m.selected {
				if idx < len(m.hits) {
					m.deletedSet[m.hits[idx].Domain] = true
				}
			}
			return m, m.deleteBatchCmd()
		case "clear-all":
			// Clear all resets everything including the deleted set.
			m.deletedSet = make(map[string]bool)
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

	// Tab bar with hit count and clock.
	tabExtra := StyleHelpDesc.Render(fmt.Sprintf("%d hits", len(m.hits))) + " " + StyleHelpDesc.Render(formatClock())
	tabBar := renderTabBar(viewExplorer, m.width, tabExtra)

	// Build the panel title from filter/sort/count info.
	panelTitle := m.buildPanelTitle()

	// Table rendered inside the panel.
	tableView := m.table.View()

	// Wrap the table in a titled panel.
	contentPanel := renderTitledPanel(panelTitle, tableView, m.width)

	// Help bar or confirmation overlay (confirmation replaces help bar).
	var bottomBar string
	if m.confirmAction != "" {
		bottomBar = m.renderConfirmPrompt()
	} else {
		bottomBar = m.renderHelpBar()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabBar,
		contentPanel,
		bottomBar,
	)
}

// buildPanelTitle constructs the title string for the explorer panel border.
func (m ExplorerModel) buildPanelTitle() string {
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

	var filterStr string
	if len(parts) > 0 {
		filterStr = strings.Join(parts, " | ")
	} else {
		filterStr = "no filters"
	}
	if m.loading {
		filterStr = "loading..."
	}

	sortLabel := fmt.Sprintf("sort:%s %s", explorerColumns[m.sortCol], m.sortDir)
	hitCount := fmt.Sprintf("%d hits", len(m.hits))

	title := fmt.Sprintf("Filters: %s ── %s ── %s", filterStr, sortLabel, hitCount)

	if len(m.selected) > 0 {
		title += fmt.Sprintf(" ── %d selected", len(m.selected))
	}

	// Status text (delete feedback).
	if m.statusText != "" {
		title += " ── " + m.statusText
	}

	return title
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
	return StyleConfirmOverlay.Width(m.width - 2).Render(" " + prompt)
}

func (m ExplorerModel) renderHelpBar() string {
	sep := StyleHelpDesc.Render("  ")
	help := StyleHelpKey.Render("Tab") + StyleHelpDesc.Render("=views") + sep +
		StyleHelpKey.Render("q") + StyleHelpDesc.Render("=quit") + sep +
		StyleHelpKey.Render("s") + StyleHelpDesc.Render("=sort") + sep +
		StyleHelpKey.Render("f") + StyleHelpDesc.Render("=filter") + sep +
		StyleHelpKey.Render("b") + StyleHelpDesc.Render("=mark") + sep +
		StyleHelpKey.Render("Space") + StyleHelpDesc.Render("=select") + sep +
		StyleHelpKey.Render("d") + StyleHelpDesc.Render("=delete") + sep +
		StyleHelpKey.Render("Enter") + StyleHelpDesc.Render("=detail")
	return " " + help
}

func (m ExplorerModel) hitsToRows() []table.Row {
	rows := make([]table.Row, 0, len(m.hits))
	for i, hit := range m.hits {
		// Checkbox column -- plain text only, no ANSI in cell data.
		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = "[x]"
		}

		kw := strings.Join(hit.Keywords, ", ")
		if len(kw) > 23 {
			kw = kw[:20] + "..."
		}

		// Domain -- plain text with prefix/suffix indicators.
		dom := hit.Domain
		maxDom := 34
		if hit.Bookmarked {
			maxDom -= 2 // room for "* " prefix
		}
		if hit.IsLive {
			maxDom -= 4 // room for " [L]" suffix
		}
		if len(dom) > maxDom {
			dom = dom[:maxDom-3] + "..."
		}
		if hit.Bookmarked {
			dom = "* " + dom
		}
		if hit.IsLive {
			dom = dom + " [L]"
		}

		issuer := hit.IssuerCN
		if len(issuer) > 18 {
			issuer = issuer[:15] + "..."
		}
		ts := hit.CreatedAt.Format("2006-01-02 15:04:05")

		// Plain text cells -- no ANSI codes. Table handles alignment.
		rows = append(rows, table.Row{
			checkbox,
			string(hit.Severity),
			fmt.Sprintf("%d", hit.Score),
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
			return deleteStatusMsg{Success: false, Count: 0, Err: err}
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
			return deleteStatusMsg{Success: false, Count: 0, Err: err}
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
