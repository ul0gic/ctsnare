package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// networkSampleMinWidth is the terminal width at or above which the network view
// shows the sample-domains column. Below it the column is dropped so the IP,
// counts, and provider stay readable.
const networkSampleMinWidth = 100

// NetworkModel lists infrastructure clusters — groups of flagged domains that
// share a resolved IP. It mirrors ExplorerModel's structure: a focused table
// loaded asynchronously from the store, with Enter drilling into the explorer
// filtered to the selected cluster's member domains.
type NetworkModel struct {
	table    table.Model
	clusters []domain.NetworkCluster
	loading  bool
	store    domain.Store
	width    int
	height   int
	ready    bool
	showSamp bool // include the sample-domains column (set by width on resize)
}

// NewNetworkModel creates a new network-clusters view.
func NewNetworkModel(store domain.Store) NetworkModel {
	t := table.New(
		table.WithColumns(networkTableColumns(false)),
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

	return NetworkModel{
		table:    t,
		clusters: make([]domain.NetworkCluster, 0),
		store:    store,
	}
}

// Init returns the initial command for the network model.
func (m NetworkModel) Init() tea.Cmd {
	return m.loadClustersCmd()
}

// Update handles messages for the network model.
func (m NetworkModel) Update(msg tea.Msg) (NetworkModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg), nil

	case ClustersLoadedMsg:
		return m.handleClustersLoaded(msg), nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// resize recomputes the table dimensions for a new terminal size and toggles the
// sample-domains column when the width crosses the disclosure threshold.
func (m NetworkModel) resize(msg tea.WindowSizeMsg) NetworkModel {
	m.width = msg.Width
	m.height = msg.Height
	// Layout matches the explorer: tabBar(3) + panel borders(2) + header(2) + helpBar(1) = 8 lines chrome.
	tableHeight := m.height - 8
	if tableHeight < 3 {
		tableHeight = 3
	}

	showSamp := m.width >= networkSampleMinWidth
	if showSamp != m.showSamp {
		m.showSamp = showSamp
		m.table.SetColumns(networkTableColumns(showSamp))
		m.table.SetRows(m.clustersToRows())
	}

	m.table.SetWidth(m.width - 2)
	m.table.SetHeight(tableHeight)
	m.ready = true
	return m
}

// handleClustersLoaded applies a freshly loaded cluster set.
func (m NetworkModel) handleClustersLoaded(msg ClustersLoadedMsg) NetworkModel {
	m.clusters = msg.Clusters
	m.loading = false
	m.table.SetRows(m.clustersToRows())
	return m
}

// handleKey routes a key press to the matching network action.
func (m NetworkModel) handleKey(msg tea.KeyMsg) (NetworkModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.drillIntoCluster()
	case "r":
		m.loading = true
		return m, m.loadClustersCmd()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// drillIntoCluster emits a ShowClusterMsg for the cluster under the cursor, so
// the app can switch to the explorer filtered to that IP's member domains.
func (m NetworkModel) drillIntoCluster() (NetworkModel, tea.Cmd) {
	row := m.table.Cursor()
	if row < 0 || row >= len(m.clusters) {
		return m, nil
	}
	ip := m.clusters[row].IP
	return m, func() tea.Msg { return ShowClusterMsg{IP: ip} }
}

// reload re-reads clusters from the store. Exposed for the app to call when the
// network view is (re)entered.
func (m NetworkModel) reload() (NetworkModel, tea.Cmd) {
	m.loading = true
	return m, m.loadClustersCmd()
}

// View renders the network model as a string.
func (m NetworkModel) View() string {
	if !m.ready {
		return "Initializing network view..."
	}

	tabExtra := StyleHelpDesc.Render(fmt.Sprintf("%d clusters", len(m.clusters))) + " " + StyleHelpDesc.Render(formatClock())
	tabBar := renderTabBar(viewNetwork, m.width, tabExtra)

	title := m.buildPanelTitle()

	tableView := m.table.View()
	if !m.loading && len(m.clusters) == 0 {
		tableView = m.renderEmptyState()
	}
	contentPanel := renderTitledPanel(title, tableView, m.width)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabBar,
		contentPanel,
		m.renderHelpBar(),
	)
}

// buildPanelTitle constructs the title string for the network panel border.
func (m NetworkModel) buildPanelTitle() string {
	status := "shared-IP clusters (≥2 domains, CDN edges excluded)"
	if m.loading {
		status = "loading..."
	}
	return fmt.Sprintf("Infrastructure ── %s ── %d clusters", status, len(m.clusters))
}

// renderEmptyState returns a centered hint shown when there are no clusters.
func (m NetworkModel) renderEmptyState() string {
	msg := "No infrastructure clusters yet. Clusters appear once enrichment resolves" +
		" two or more flagged domains to the same dedicated IP."
	hint := StyleHelpDesc.Render(msg)
	return lipgloss.Place(
		m.table.Width(), m.table.Height(),
		lipgloss.Center, lipgloss.Center,
		hint,
	)
}

func (m NetworkModel) renderHelpBar() string {
	sep := StyleHelpDesc.Render("  ")
	kv := func(k, v string) string {
		return StyleHelpKey.Render(k) + StyleHelpDesc.Render(v)
	}
	help := kv("Tab", "=views") + sep +
		kv("Enter", "=domains") + sep +
		kv("j/k", "=move") + sep +
		kv("r", "=reload") + sep +
		kv("?", "=help") + sep +
		kv("q", "=quit")
	return " " + help
}

// networkTableColumns returns the cluster table column set, optionally including
// the sample-domains column. Column order must match the cell order in
// clusterToRow.
func networkTableColumns(showSamp bool) []table.Column {
	cols := []table.Column{
		{Title: "IP", Width: 17},
		{Title: "Domains", Width: 8},
		{Title: "Live", Width: 6},
		{Title: "Provider", Width: 14},
		{Title: "Score", Width: 6},
	}
	if showSamp {
		cols = append(cols, table.Column{Title: "Sample Domains", Width: 40})
	}
	return cols
}

func (m NetworkModel) clustersToRows() []table.Row {
	rows := make([]table.Row, 0, len(m.clusters))
	for _, c := range m.clusters {
		rows = append(rows, m.clusterToRow(c))
	}
	return rows
}

// clusterToRow formats a single cluster into a table row.
func (m NetworkModel) clusterToRow(c domain.NetworkCluster) table.Row {
	provider := c.HostingProvider
	if provider == "" {
		provider = "—"
	}
	row := table.Row{
		c.IP,
		strconv.Itoa(c.DomainCount),
		fmt.Sprintf("%d/%d", c.LiveCount, c.DomainCount),
		truncate(provider, 14),
		strconv.Itoa(c.MaxScore),
	}
	if m.showSamp {
		row = append(row, truncate(strings.Join(c.SampleDomains, ", "), 40))
	}
	return row
}

func (m NetworkModel) loadClustersCmd() tea.Cmd {
	if m.store == nil {
		return func() tea.Msg { return ClustersLoadedMsg{Clusters: nil} }
	}
	store := m.store
	return func() tea.Msg {
		clusters, err := store.NetworkClusters(context.Background())
		if err != nil {
			return ClustersLoadedMsg{Clusters: nil}
		}
		return ClustersLoadedMsg{Clusters: clusters}
	}
}
