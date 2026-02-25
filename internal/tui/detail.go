package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// DetailModel displays the full details of a single hit record.
type DetailModel struct {
	hit      domain.Hit
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewDetailModel creates a new detail view for a specific hit.
func NewDetailModel(hit domain.Hit) DetailModel {
	return DetailModel{
		hit: hit,
	}
}

// Init returns the initial command for the detail model.
func (m DetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the detail model.
func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "q" {
			return m, func() tea.Msg {
				return SwitchViewMsg{View: 1}
			}
		}
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

// View renders the detail model as a string.
func (m DetailModel) View() string {
	if !m.ready {
		return "Initializing detail view..."
	}

	// Tab bar.
	tabBar := renderTabBar(viewDetail, m.width, "")

	// Build the panel title with domain, severity, and score.
	panelTitle := m.buildPanelTitle()

	// Viewport content inside a titled panel.
	contentPanel := renderTitledPanel(panelTitle, m.viewport.View(), m.width)

	// Help bar.
	sep := StyleHelpDesc.Render("  ")
	helpBar := " " + StyleHelpKey.Render("Esc") + StyleHelpDesc.Render("=back") + sep +
		StyleHelpKey.Render("j/k") + StyleHelpDesc.Render("=scroll")

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, contentPanel, helpBar)
}

// buildPanelTitle constructs the detail panel's title with domain, severity, and score.
func (m DetailModel) buildPanelTitle() string {
	var parts []string

	// Bookmark indicator.
	if m.hit.Bookmarked {
		parts = append(parts, StyleBookmarked.Render("*"))
	}

	// Domain name colored by severity.
	sevStyle := SeverityStyle(string(m.hit.Severity))
	parts = append(parts, sevStyle.Render(m.hit.Domain))

	title := strings.Join(parts, " ")

	// Severity and score in the title.
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
	sepWidth := contentWidth - 2 // a bit of padding

	// Certificate section.
	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Certificate") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	b.WriteString(renderField("Issuer Org", m.hit.Issuer))
	b.WriteString(renderField("Issuer CN", m.hit.IssuerCN))
	if !m.hit.CertNotBefore.IsZero() {
		b.WriteString(renderField("Cert Not Before", m.hit.CertNotBefore.Format("2006-01-02 15:04:05 UTC")))
	}

	// Scoring section.
	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Scoring") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	if len(m.hit.Keywords) > 0 {
		b.WriteString(renderField("Keywords", strings.Join(m.hit.Keywords, ", ")))
	} else {
		b.WriteString(renderField("Keywords", "(none)"))
	}
	b.WriteString(renderField("CT Log", m.hit.CTLog))
	b.WriteString(renderField("Profile", m.hit.Profile))
	b.WriteString(renderField("Session", m.hit.Session))

	// SANs section.
	b.WriteString("\n")
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("SANs") + "\n")
	b.WriteString("  " + renderDottedSep(sepWidth) + "\n")
	if len(m.hit.SANDomains) > 0 {
		for _, san := range m.hit.SANDomains {
			b.WriteString(fmt.Sprintf("    %s\n", san))
		}
	} else {
		b.WriteString("    (none)\n")
	}

	// Enrichment data section -- only shown if enrichment has run.
	if !m.hit.LiveCheckedAt.IsZero() {
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
			b.WriteString(renderField("HTTP Status", fmt.Sprintf("%d", m.hit.HTTPStatus)))
		}

		b.WriteString(renderField("Last Checked", m.hit.LiveCheckedAt.Format("2006-01-02 15:04:05")))
	}

	// Timestamps at the bottom.
	b.WriteString("\n")
	if !m.hit.CreatedAt.IsZero() {
		b.WriteString(renderField("First Seen", m.hit.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if !m.hit.UpdatedAt.IsZero() {
		b.WriteString(renderField("Last Updated", m.hit.UpdatedAt.Format("2006-01-02 15:04:05")))
	}

	return b.String()
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
