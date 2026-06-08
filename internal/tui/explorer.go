package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	keepSelection bool   // preserve selection across the next reload (e.g. sort)
	confirmAction string // empty, "delete-single", "delete-batch", "clear-all"
	confirmDomain string // domain for single delete confirmation
	// deletedSet holds domains deleted since the last reload, filtered out of the
	// next HitsLoadedMsg so an in-flight poller re-insert cannot resurrect a row
	// the user just deleted. It is bounded to a single reload cycle: every
	// explicit reload (sort, filter apply, tab-in, r) clears it, because the
	// freshly loaded set already reflects deletions committed to the DB. A domain
	// is therefore only suppressed until the next reload — if it is legitimately
	// re-observed and re-stored later, it reappears on the next load.
	deletedSet map[string]bool
	statusText string // brief status message shown in filter bar
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
		return m.resize(msg), nil

	case HitsLoadedMsg:
		return m.handleHitsLoaded(msg), nil

	case DeleteHitsMsg:
		// Hits were deleted -- reload.
		m.selected = make(map[int]bool)
		m.loading = true
		return m, m.loadHitsCmd()

	case deleteStatusMsg:
		return m.applyDeleteStatus(msg), nil

	case BookmarkToggleMsg:
		return m.applyBookmarkToggle(msg), nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// resize recomputes the table dimensions to fit the new terminal size.
func (m ExplorerModel) resize(msg tea.WindowSizeMsg) ExplorerModel {
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
	return m
}

// applyDeleteStatus records the outcome of a delete operation in the status bar.
func (m ExplorerModel) applyDeleteStatus(msg deleteStatusMsg) ExplorerModel {
	if msg.Success {
		m.statusText = fmt.Sprintf("Deleted %d hit(s)", msg.Count)
	} else {
		m.statusText = fmt.Sprintf("Delete failed: %v", msg.Err)
	}
	return m
}

// applyBookmarkToggle updates the local bookmark state for a domain and
// refreshes the affected row.
func (m ExplorerModel) applyBookmarkToggle(msg BookmarkToggleMsg) ExplorerModel {
	for i := range m.hits {
		if m.hits[i].Domain == msg.Domain {
			m.hits[i].Bookmarked = msg.Bookmarked
			break
		}
	}
	m.table.SetRows(m.hitsToRows())
	return m
}

// handleHitsLoaded applies a freshly loaded hit set, filtering out recently
// deleted domains and remapping the selection by domain identity when a reload
// was requested with keepSelection set.
func (m ExplorerModel) handleHitsLoaded(msg HitsLoadedMsg) ExplorerModel {
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
	return m
}

// handleKey routes a key press to the matching explorer action. The
// confirmation overlay takes precedence over all other bindings.
func (m ExplorerModel) handleKey(msg tea.KeyMsg) (ExplorerModel, tea.Cmd) {
	// Handle confirmation overlay first.
	if m.confirmAction != "" {
		return m.handleConfirm(msg)
	}

	// Clear status message on any key press.
	m.statusText = ""

	// Clear-all (C) is the single most destructive action; route it through the
	// keymap binding so the help overlay and the handler share one source of truth.
	if key.Matches(msg, m.keys.Clear) {
		return m.handleDeleteKey("C"), nil
	}

	if handled, model, cmd := m.handleActionKey(msg.String()); handled {
		return model, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// handleActionKey dispatches the explorer's non-confirmation action keys. It
// returns handled=false when the key should fall through to the table.
func (m ExplorerModel) handleActionKey(key string) (handled bool, model ExplorerModel, cmd tea.Cmd) {
	switch key {
	case "s":
		return true, m.cycleSort(), m.loadHitsCmd()

	case "enter":
		model, cmd = m.showDetailCmd()
		return true, model, cmd

	case "r":
		model, cmd = m.reload()
		return true, model, cmd

	case " ", "a", "A": // selection keys
		model, cmd = m.handleSelectionKey(key)
		return true, model, cmd

	case "d", "D": // destructive keys (open a confirmation)
		return true, m.handleDeleteKey(key), nil

	case "b": // bookmark toggle
		model, cmd = m.bookmarkCurrent()
		return true, model, cmd
	}
	return false, m, nil
}

// showDetailCmd emits a ShowDetailMsg for the row under the cursor, if any.
func (m ExplorerModel) showDetailCmd() (ExplorerModel, tea.Cmd) {
	if row, ok := m.currentRow(); ok {
		hit := m.hits[row]
		return m, func() tea.Msg { return ShowDetailMsg{Hit: hit} }
	}
	return m, nil
}

// reload triggers an explicit reload, clearing the per-cycle deleted-domain
// guard and any transient status text.
func (m ExplorerModel) reload() (ExplorerModel, tea.Cmd) {
	m.deletedSet = make(map[string]bool)
	m.statusText = ""
	m.loading = true
	return m, m.loadHitsCmd()
}

// bookmarkCurrent toggles the bookmark on the row under the cursor, if any.
func (m ExplorerModel) bookmarkCurrent() (ExplorerModel, tea.Cmd) {
	if row, ok := m.currentRow(); ok {
		return m, m.bookmarkToggleCmd(row)
	}
	return m, nil
}

// handleSelectionKey handles the row-selection bindings: toggle (space),
// select-all (a), and deselect-all (A).
func (m ExplorerModel) handleSelectionKey(key string) (ExplorerModel, tea.Cmd) {
	switch key {
	case " ":
		return m.toggleSelect()
	case "a":
		for i := range m.hits {
			m.selected[i] = true
		}
	case "A":
		m.selected = make(map[int]bool)
	}
	m.table.SetRows(m.hitsToRows())
	return m, nil
}

// handleDeleteKey arms a confirmation overlay for the destructive bindings:
// delete-single (d), delete-batch (D), and clear-all (C).
func (m ExplorerModel) handleDeleteKey(key string) ExplorerModel {
	switch key {
	case "d":
		if row, ok := m.currentRow(); ok {
			m.confirmAction = "delete-single"
			m.confirmDomain = m.hits[row].Domain
		}
	case "D":
		if len(m.selected) > 0 {
			m.confirmAction = "delete-batch"
		}
	case "C":
		m.confirmAction = "clear-all"
	}
	return m
}

// currentRow returns the index of the row under the cursor and whether it is
// within bounds of the current hit set.
func (m ExplorerModel) currentRow() (int, bool) {
	row := m.table.Cursor()
	return row, row >= 0 && row < len(m.hits)
}

// cycleSort advances to the next sort column, flips the sort direction, and
// marks the model for a selection-preserving reload.
func (m ExplorerModel) cycleSort() ExplorerModel {
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
	// A sort reload re-reads committed state; the deleted-domain guard is no
	// longer needed and must not leak into the next session.
	m.deletedSet = make(map[string]bool)
	return m
}

// toggleSelect flips the selection state of the row under the cursor and
// advances the cursor down one row.
func (m ExplorerModel) toggleSelect() (ExplorerModel, tea.Cmd) {
	row, ok := m.currentRow()
	if !ok {
		return m, nil
	}
	if m.selected[row] {
		delete(m.selected, row)
	} else {
		m.selected[row] = true
	}
	m.table.SetRows(m.hitsToRows())
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(tea.KeyMsg{Type: tea.KeyDown})
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

	// Table rendered inside the panel, or an empty-state hint when there are
	// no rows to show.
	tableView := m.table.View()
	if !m.loading && len(m.hits) == 0 {
		tableView = m.renderEmptyState()
	}

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
	parts := m.filterParts()

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

// filterParts renders the active filter fields as human-readable label segments.
func (m ExplorerModel) filterParts() []string {
	var parts []string
	if m.filter.Keyword != "" {
		parts = append(parts, "keyword:"+m.filter.Keyword)
	}
	if m.filter.ScoreMin > 0 {
		parts = append(parts, fmt.Sprintf("score>=%d", m.filter.ScoreMin))
	}
	if m.filter.Severity != "" {
		parts = append(parts, "severity:"+m.filter.Severity)
	}
	if m.filter.TLD != "" {
		parts = append(parts, "tld:"+m.filter.TLD)
	}
	if m.filter.Session != "" {
		parts = append(parts, "session:"+m.filter.Session)
	}
	if m.filter.LiveOnly {
		parts = append(parts, "live:yes")
	}
	if m.filter.Bookmarked != nil {
		if *m.filter.Bookmarked {
			parts = append(parts, "bookmarked:yes")
		} else {
			parts = append(parts, "bookmarked:no")
		}
	}
	if m.filter.BaseDomain != "" {
		parts = append(parts, "base:"+m.filter.BaseDomain)
	}
	return parts
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
	// A filter-driven reload re-reads committed state, so the per-cycle
	// deleted-domain guard can be dropped here too.
	m.deletedSet = make(map[string]bool)
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

// renderEmptyState returns a centered hint shown when the table has no rows.
// It distinguishes "no data yet" from "no matches for the active filter" so the
// user can tell whether the database is empty or their filter excluded
// everything. The block is padded to roughly the table height to keep the panel
// from collapsing.
func (m ExplorerModel) renderEmptyState() string {
	var msg string
	if len(m.filterParts()) > 0 {
		msg = "No hits match the current filter. Press f to adjust, or ctrl+l in the filter to clear it."
	} else {
		msg = "No hits found. Start watching to populate (ctsnare watch)."
	}

	hint := StyleHelpDesc.Render(msg)
	body := lipgloss.Place(
		m.table.Width(), m.table.Height(),
		lipgloss.Center, lipgloss.Center,
		hint,
	)
	return body
}

func (m ExplorerModel) renderHelpBar() string {
	sep := StyleHelpDesc.Render("  ")
	kv := func(k, v string) string {
		return StyleHelpKey.Render(k) + StyleHelpDesc.Render(v)
	}
	// Lead with the everyday keys, then the destructive batch keys (notably C,
	// which wipes the whole DB), then ? for the full key list.
	help := kv("Tab", "=views") + sep +
		kv("s", "=sort") + sep +
		kv("f", "=filter") + sep +
		kv("/", "=search") + sep +
		kv("Space", "=select") + sep +
		kv("b", "=mark") + sep +
		kv("d", "=del") + sep +
		kv("D", "=del-sel") + sep +
		StyleHighSeverity.Render("C") + StyleHelpDesc.Render("=clear-all") + sep +
		kv("r", "=reload") + sep +
		kv("?", "=help") + sep +
		kv("q", "=quit")
	return " " + help
}

func (m ExplorerModel) hitsToRows() []table.Row {
	rows := make([]table.Row, 0, len(m.hits))
	for i, hit := range m.hits {
		rows = append(rows, m.hitToRow(i, hit))
	}
	return rows
}

// hitToRow formats a single hit into a styled table row. Index i is used to
// reflect the row's selection state in the checkbox column.
func (m ExplorerModel) hitToRow(i int, hit domain.Hit) table.Row {
	// Checkbox column -- plain text only, no ANSI in cell data.
	checkbox := "[ ]"
	if m.selected[i] {
		checkbox = "[x]"
	}

	kw := truncate(strings.Join(hit.Keywords, ", "), 23)
	dom := formatDomainCell(hit)
	issuer := truncate(hit.IssuerCN, 18)
	ts := hit.CreatedAt.Format("2006-01-02 15:04:05")

	// Color cells by severity.
	sevStyle := SeverityStyle(string(hit.Severity))
	sevText := sevStyle.Render(string(hit.Severity))
	scoreText := sevStyle.Render(strconv.Itoa(hit.Score))

	// Domain colored by severity, or green if live.
	domText := sevStyle.Render(dom)
	if hit.IsLive {
		domText = StyleLiveDomain.Render(dom)
	}

	return table.Row{checkbox, sevText, scoreText, domText, kw, issuer, hit.Session, ts}
}

// truncate shortens s to at most maxLen runes, appending an ellipsis when cut.
// It operates on runes, not bytes, so multibyte (IDN/homograph) domains are
// never split mid-rune into invalid UTF-8.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}

// formatDomainCell renders a domain with bookmark/live indicators, truncating
// the domain so the decorated string fits the column width.
func formatDomainCell(hit domain.Hit) string {
	dom := hit.Domain
	maxDom := 34
	if hit.Bookmarked {
		maxDom -= 2 // room for "* " prefix
	}
	if hit.IsLive {
		maxDom -= 4 // room for " [L]" suffix
	}
	dom = truncate(dom, maxDom)
	if hit.Bookmarked {
		dom = "* " + dom
	}
	if hit.IsLive {
		dom += " [L]"
	}
	return dom
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
