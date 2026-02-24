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
		contentHeight := m.height - 4
		if contentHeight < 1 {
			contentHeight = 1
		}
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

	title := StyleTitle.Render(fmt.Sprintf("Hit Detail: %s", m.hit.Domain))
	header := StyleHeader.Width(m.width).Render(title)
	help := StyleStatusBar.Width(m.width).Render(
		StyleHelpKey.Render("esc") + StyleHelpDesc.Render(" back") + "  " +
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" scroll"),
	)

	panel := StyleBorder.Width(m.width - 2).Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, panel, help)
}

func (m DetailModel) renderContent() string {
	var b strings.Builder

	sevStyle := SeverityStyle(string(m.hit.Severity))

	b.WriteString(renderField("Domain", m.hit.Domain))
	b.WriteString(renderField("Score", fmt.Sprintf("%d", m.hit.Score)))
	b.WriteString(renderField("Severity", sevStyle.Render(string(m.hit.Severity))))
	b.WriteByte('\n')

	b.WriteString(renderField("Issuer Org", m.hit.Issuer))
	b.WriteString(renderField("Issuer CN", m.hit.IssuerCN))
	if !m.hit.CertNotBefore.IsZero() {
		b.WriteString(renderField("Cert Not Before", m.hit.CertNotBefore.Format("2006-01-02 15:04:05 UTC")))
	}
	b.WriteByte('\n')

	b.WriteString(renderField("CT Log", m.hit.CTLog))
	b.WriteString(renderField("Profile", m.hit.Profile))
	b.WriteString(renderField("Session", m.hit.Session))
	b.WriteByte('\n')

	b.WriteString(StyleTitle.Render("Matched Keywords"))
	b.WriteByte('\n')
	if len(m.hit.Keywords) > 0 {
		for _, kw := range m.hit.Keywords {
			b.WriteString(fmt.Sprintf("  - %s\n", kw))
		}
	} else {
		b.WriteString("  (none)\n")
	}
	b.WriteByte('\n')

	b.WriteString(StyleTitle.Render("Subject Alternative Names"))
	b.WriteByte('\n')
	if len(m.hit.SANDomains) > 0 {
		for _, san := range m.hit.SANDomains {
			b.WriteString(fmt.Sprintf("  - %s\n", san))
		}
	} else {
		b.WriteString("  (none)\n")
	}
	b.WriteByte('\n')

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
