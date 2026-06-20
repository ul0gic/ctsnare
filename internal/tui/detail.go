package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// DetailModel displays the full details of a single hit record.
type DetailModel struct {
	hit            domain.Hit
	store          domain.Store
	viewport       viewport.Model
	width          int
	height         int
	ready          bool
	subdomainCount int  // number of related subdomains (0 means not loaded yet)
	countLoaded    bool // true after the async count has been received
}

// NewDetailModel creates a detail view for a hit. A nil store hides the
// related-subdomains section.
func NewDetailModel(hit domain.Hit, store domain.Store) DetailModel {
	return DetailModel{
		hit:   hit,
		store: store,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return m.loadSubdomainCountCmd()
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg), nil

	case SubdomainCountMsg:
		if msg.BaseDomain == m.hit.BaseDomain {
			m.subdomainCount = msg.Count
			m.countLoaded = true
			if m.ready {
				m.viewport.SetContent(m.renderContent())
			}
		}
		return m, nil

	case tea.KeyMsg:
		if handled, model, keyCmd := m.handleKey(msg); handled {
			return model, keyCmd
		}
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

// resize recomputes the detail viewport dimensions for a new terminal size,
// creating the viewport on first sizing.
func (m DetailModel) resize(msg tea.WindowSizeMsg) DetailModel {
	m.width = msg.Width
	m.height = msg.Height
	// Layout: tabBar(3) + panel top/bottom borders(2) + helpBar(1) = 6 lines of chrome.
	contentHeight := m.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}
	// Content width is inside the panel borders (2 chars).
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	if !m.ready {
		m.viewport = viewport.New(contentWidth, contentHeight)
		m.ready = true
	} else {
		m.viewport.Width = contentWidth
		m.viewport.Height = contentHeight
	}
	m.viewport.SetContent(m.renderContent())
	return m
}

// handleKey processes detail-view bindings (esc/q back, enter drills into
// subdomains), returning handled=false to fall through to the scrolling viewport.
func (m DetailModel) handleKey(msg tea.KeyMsg) (handled bool, model DetailModel, cmd tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return true, m, func() tea.Msg { return SwitchViewMsg{View: viewExplorer} }
	case "enter":
		if m.countLoaded && m.subdomainCount > 1 && m.hit.BaseDomain != "" {
			baseDomain := m.hit.BaseDomain
			fromDomain := m.hit.Domain
			return true, m, func() tea.Msg {
				return ShowSubdomainsMsg{BaseDomain: baseDomain, FromDomain: fromDomain}
			}
		}
	}
	return false, m, nil
}

func (m DetailModel) View() string {
	if !m.ready {
		return "Initializing detail view..."
	}

	tabBar := renderTabBar(viewDetail, m.width, "")
	panelTitle := m.buildPanelTitle()
	contentPanel := renderTitledPanel(panelTitle, m.viewport.View(), m.width)

	sep := StyleHelpDesc.Render("  ")
	helpBar := " " + StyleHelpKey.Render("Esc") + StyleHelpDesc.Render("=back") + sep +
		StyleHelpKey.Render("j/k") + StyleHelpDesc.Render("=scroll")

	if m.countLoaded && m.subdomainCount > 1 {
		helpBar += sep + StyleHelpKey.Render("Enter") + StyleHelpDesc.Render("=subdomains")
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, contentPanel, helpBar)
}

// buildPanelTitle constructs the detail panel's title with domain, severity, and score.
func (m DetailModel) buildPanelTitle() string {
	var parts []string

	if m.hit.Bookmarked {
		parts = append(parts, StyleBookmarked.Render("*"))
	}

	sevStyle := SeverityStyle(string(m.hit.Severity))
	parts = append(parts, sevStyle.Render(m.hit.Domain))

	title := strings.Join(parts, " ")

	sevTag := sevStyle.Render(string(m.hit.Severity))
	scoreTag := sevStyle.Render(fmt.Sprintf("Score: %d", m.hit.Score))

	return title + " ── " + sevTag + " ── " + scoreTag
}

// renderDottedSep renders a dotted separator line at the given width.
func renderDottedSep(width int) string {
	if width < 1 {
		width = 1
	}
	return StyleDottedSep.Render(strings.Repeat("┄", width))
}

func (m DetailModel) renderContent() string {
	var b strings.Builder
	contentWidth := m.width - 4 // inside panel borders
	if contentWidth < 20 {
		contentWidth = 20
	}
	sepWidth := contentWidth - 2

	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Certificate") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	b.WriteString(renderField("Issuer Org", m.hit.Issuer))
	b.WriteString(renderField("Issuer CN", m.hit.IssuerCN))
	if !m.hit.CertNotBefore.IsZero() {
		b.WriteString(renderField("Cert Not Before", m.hit.CertNotBefore.Format("2006-01-02 15:04:05 UTC")))
	}

	m.renderScoring(&b, sepWidth)
	m.renderSANs(&b, sepWidth)
	m.renderEnrichment(&b, sepWidth)

	if m.countLoaded && m.subdomainCount > 1 && m.hit.BaseDomain != "" {
		b.WriteString("\n")
		b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Related Subdomains") + "\n")
		b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
		b.WriteString(renderField("Base Domain", m.hit.BaseDomain))
		countStr := fmt.Sprintf("%d (Enter to view)", m.subdomainCount)
		b.WriteString(renderField("Subdomains", countStr))
	}

	b.WriteString("\n")
	if !m.hit.CreatedAt.IsZero() {
		b.WriteString(renderField("First Seen", m.hit.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if !m.hit.UpdatedAt.IsZero() {
		b.WriteString(renderField("Last Updated", m.hit.UpdatedAt.Format("2006-01-02 15:04:05")))
	}

	return b.String()
}

// renderScoring writes the scoring section (keywords, signals, category, and
// the CT-log/profile/session provenance fields) into b.
func (m DetailModel) renderScoring(b *strings.Builder, sepWidth int) {
	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Scoring") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	if len(m.hit.Keywords) > 0 {
		b.WriteString(renderField("Keywords", strings.Join(m.hit.Keywords, ", ")))
	} else {
		b.WriteString(renderField("Keywords", "(none)"))
	}
	if len(m.hit.Signals) > 0 {
		b.WriteString(renderField("Signals", strings.Join(m.hit.Signals, ", ")))
	}
	if m.hit.Category != "" {
		b.WriteString(renderField("Category", m.hit.Category))
	}
	b.WriteString(renderField("CT Log", m.hit.CTLog))
	b.WriteString(renderField("Profile", m.hit.Profile))
	b.WriteString(renderField("Session", m.hit.Session))
}

// renderSANs writes the Subject Alternative Names section into b.
func (m DetailModel) renderSANs(b *strings.Builder, sepWidth int) {
	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("SANs") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	if len(m.hit.SANDomains) == 0 {
		b.WriteString("    (none)\n")
		return
	}
	for _, san := range m.hit.SANDomains {
		fmt.Fprintf(b, "    %s\n", san)
	}
}

// renderEnrichment writes the DNS/HTTP enrichment section into b. It is a no-op
// until enrichment has run for the hit (LiveCheckedAt is set).
func (m DetailModel) renderEnrichment(b *strings.Builder, sepWidth int) {
	if m.hit.LiveCheckedAt.IsZero() {
		return
	}

	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Enrichment") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")

	liveStr := lipgloss.NewStyle().Foreground(colorHighSeverity).Render("No")
	if m.hit.IsLive {
		liveStr = StyleLiveDomain.Render("Yes")
	}
	b.WriteString(renderField("Live", liveStr))

	if len(m.hit.ResolvedIPs) > 0 {
		b.WriteString(renderField("Resolved IPs", strings.Join(m.hit.ResolvedIPs, ", ")))
	} else {
		b.WriteString(renderField("Resolved IPs", "(none)"))
	}

	if m.hit.HostingProvider != "" {
		b.WriteString(renderField("Hosting", m.hit.HostingProvider))
	}

	if m.hit.HTTPStatus > 0 {
		b.WriteString(renderField("HTTP Status", strconv.Itoa(m.hit.HTTPStatus)))
	}

	b.WriteString(renderField("Last Checked", m.hit.LiveCheckedAt.Format("2006-01-02 15:04:05")))
}

func renderField(label, value string) string {
	if value == "" {
		value = "(empty)"
	}
	return fmt.Sprintf("  %s  %s\n",
		StyleHelpKey.Width(16).Render(label+":"),
		value,
	)
}

// loadSubdomainCountCmd returns a tea.Cmd that asynchronously queries the
// store for the number of hits sharing this hit's base domain.
func (m DetailModel) loadSubdomainCountCmd() tea.Cmd {
	if m.store == nil || m.hit.BaseDomain == "" {
		return func() tea.Msg {
			return SubdomainCountMsg{BaseDomain: m.hit.BaseDomain, Count: 0}
		}
	}
	store := m.store
	baseDomain := m.hit.BaseDomain
	return func() tea.Msg {
		count, err := store.CountByBaseDomain(context.Background(), baseDomain)
		if err != nil {
			return SubdomainCountMsg{BaseDomain: baseDomain, Count: 0}
		}
		return SubdomainCountMsg{BaseDomain: baseDomain, Count: count}
	}
}
