package tui

import (
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	filterFieldKeyword = iota
	filterFieldScoreMin
	filterFieldSeverity
	filterFieldTimeRange
	filterFieldSession
	filterFieldCount
)

var severityOptions = []string{"", "HIGH", "MED", "LOW"}
var timeRangeOptions = []string{"", "1h", "6h", "12h", "24h", "7d"}

// FilterAppliedMsg is sent when the user applies filter settings.
type FilterAppliedMsg struct {
	Filter domain.QueryFilter
}

// FilterCancelledMsg is sent when the user cancels the filter overlay.
type FilterCancelledMsg struct{}

// FilterModel provides an input overlay for building query filters.
type FilterModel struct {
	inputs       []textinput.Model
	activeField  int
	severityIdx  int
	timeRangeIdx int
	width        int
	height       int
}

// NewFilterModel creates a new filter input overlay.
func NewFilterModel() FilterModel {
	inputs := make([]textinput.Model, filterFieldCount)

	keyword := textinput.New()
	keyword.Placeholder = "keyword filter"
	keyword.Prompt = "Keyword:    "
	keyword.CharLimit = 64
	inputs[filterFieldKeyword] = keyword

	scoreMin := textinput.New()
	scoreMin.Placeholder = "0"
	scoreMin.Prompt = "Min Score:  "
	scoreMin.CharLimit = 3
	inputs[filterFieldScoreMin] = scoreMin

	severity := textinput.New()
	severity.Placeholder = "all"
	severity.Prompt = "Severity:   "
	severity.CharLimit = 4
	inputs[filterFieldSeverity] = severity

	timeRange := textinput.New()
	timeRange.Placeholder = "all"
	timeRange.Prompt = "Time Range: "
	timeRange.CharLimit = 3
	inputs[filterFieldTimeRange] = timeRange

	session := textinput.New()
	session.Placeholder = "all sessions"
	session.Prompt = "Session:    "
	session.CharLimit = 64
	inputs[filterFieldSession] = session

	inputs[filterFieldKeyword].Focus()

	return FilterModel{
		inputs:      inputs,
		activeField: filterFieldKeyword,
	}
}

// Init returns the initial command for the filter model.
func (m FilterModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the filter model.
func (m FilterModel) Update(msg tea.Msg) (FilterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return FilterCancelledMsg{} }

		case "enter":
			return m, func() tea.Msg {
				return FilterAppliedMsg{Filter: m.buildFilter()}
			}

		case "tab", "down":
			m.inputs[m.activeField].Blur()
			m.activeField = (m.activeField + 1) % filterFieldCount
			return m, m.inputs[m.activeField].Focus()

		case "shift+tab", "up":
			m.inputs[m.activeField].Blur()
			m.activeField = (m.activeField - 1 + filterFieldCount) % filterFieldCount
			return m, m.inputs[m.activeField].Focus()

		case "ctrl+l":
			for i := range m.inputs {
				m.inputs[i].Reset()
			}
			m.severityIdx = 0
			m.timeRangeIdx = 0
			return m, nil
		}

		if m.activeField == filterFieldSeverity {
			if msg.String() == "left" || msg.String() == "h" {
				m.severityIdx = (m.severityIdx - 1 + len(severityOptions)) % len(severityOptions)
				m.inputs[filterFieldSeverity].SetValue(severityOptions[m.severityIdx])
				return m, nil
			}
			if msg.String() == "right" || msg.String() == "l" {
				m.severityIdx = (m.severityIdx + 1) % len(severityOptions)
				m.inputs[filterFieldSeverity].SetValue(severityOptions[m.severityIdx])
				return m, nil
			}
		}

		if m.activeField == filterFieldTimeRange {
			if msg.String() == "left" || msg.String() == "h" {
				m.timeRangeIdx = (m.timeRangeIdx - 1 + len(timeRangeOptions)) % len(timeRangeOptions)
				m.inputs[filterFieldTimeRange].SetValue(timeRangeOptions[m.timeRangeIdx])
				return m, nil
			}
			if msg.String() == "right" || msg.String() == "l" {
				m.timeRangeIdx = (m.timeRangeIdx + 1) % len(timeRangeOptions)
				m.inputs[filterFieldTimeRange].SetValue(timeRangeOptions[m.timeRangeIdx])
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.activeField], cmd = m.inputs[m.activeField].Update(msg)
	return m, cmd
}

// View renders the filter overlay as a string.
func (m FilterModel) View() string {
	title := StyleTitle.Render("Filter Hits")

	var fields string
	for _, input := range m.inputs {
		fields += input.View() + "\n"
	}

	help := StyleHelpKey.Render("enter") + StyleHelpDesc.Render(" apply") + "  " +
		StyleHelpKey.Render("esc") + StyleHelpDesc.Render(" cancel") + "  " +
		StyleHelpKey.Render("tab") + StyleHelpDesc.Render(" next field") + "  " +
		StyleHelpKey.Render("ctrl+l") + StyleHelpDesc.Render(" clear all")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", fields, help)

	panelWidth := 50
	if m.width > 0 && m.width < panelWidth+4 {
		panelWidth = m.width - 4
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		StyleBorder.Width(panelWidth).Padding(1, 2).Render(content),
	)
}

func (m FilterModel) buildFilter() domain.QueryFilter {
	var f domain.QueryFilter

	f.Keyword = m.inputs[filterFieldKeyword].Value()

	if v := m.inputs[filterFieldScoreMin].Value(); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.ScoreMin = n
		}
	}

	f.Severity = m.inputs[filterFieldSeverity].Value()
	f.Session = m.inputs[filterFieldSession].Value()

	if v := m.inputs[filterFieldTimeRange].Value(); v != "" {
		f.Since = parseTimeRange(v)
	}

	return f
}

func parseTimeRange(s string) time.Duration {
	switch s {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "12h":
		return 12 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}
