package tui

import (
	"fmt"
	"strconv"
	"strings"
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
	filterFieldTLD
	filterFieldSession
	filterFieldBookmarked
	filterFieldLiveOnly
	filterFieldCount
)

var (
	severityOptions   = []string{"", "HIGH", "MED", "LOW"}
	timeRangeOptions  = []string{"", "1h", "6h", "12h", "24h", "7d"}
	bookmarkedOptions = []string{"", "yes", "no"}
	liveOnlyOptions   = []string{"", "yes"}
)

// FilterAppliedMsg is sent when the user applies filter settings.
type FilterAppliedMsg struct {
	Filter domain.QueryFilter
}

// FilterCancelledMsg is sent when the user cancels the filter overlay.
type FilterCancelledMsg struct{}

// FilterModel provides an input overlay for building query filters.
type FilterModel struct {
	inputs        []textinput.Model
	activeField   int
	severityIdx   int
	timeRangeIdx  int
	bookmarkedIdx int
	liveOnlyIdx   int
	width         int
	height        int
	errMsg        string // inline validation error shown on a rejected apply
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
	severity.Placeholder = "all (◄/► to cycle)"
	severity.Prompt = "Severity:   "
	severity.CharLimit = 4
	inputs[filterFieldSeverity] = severity

	timeRange := textinput.New()
	timeRange.Placeholder = "all (◄/► to cycle)"
	timeRange.Prompt = "Time Range: "
	timeRange.CharLimit = 3
	inputs[filterFieldTimeRange] = timeRange

	tld := textinput.New()
	tld.Placeholder = "e.g. xyz"
	tld.Prompt = "TLD:        "
	tld.CharLimit = 12
	inputs[filterFieldTLD] = tld

	session := textinput.New()
	session.Placeholder = "all sessions"
	session.Prompt = "Session:    "
	session.CharLimit = 64
	inputs[filterFieldSession] = session

	bookmarked := textinput.New()
	bookmarked.Placeholder = "all (◄/► to cycle)"
	bookmarked.Prompt = "Bookmarked: "
	bookmarked.CharLimit = 3
	inputs[filterFieldBookmarked] = bookmarked

	liveOnly := textinput.New()
	liveOnly.Placeholder = "all (◄/► to cycle)"
	liveOnly.Prompt = "Live Only:  "
	liveOnly.CharLimit = 3
	inputs[filterFieldLiveOnly] = liveOnly

	inputs[filterFieldKeyword].Focus()

	return FilterModel{
		inputs:      inputs,
		activeField: filterFieldKeyword,
	}
}

func (m FilterModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FilterModel) Update(msg tea.Msg) (FilterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if handled, model, cmd := m.handleKey(msg); handled {
			return model, cmd
		}
	}

	var cmd tea.Cmd
	m.inputs[m.activeField], cmd = m.inputs[m.activeField].Update(msg)
	return m, cmd
}

// handleKey processes the filter overlay's key bindings. It returns
// handled=false when the key should fall through to the active text input.
func (m FilterModel) handleKey(msg tea.KeyMsg) (handled bool, model FilterModel, cmd tea.Cmd) {
	switch msg.String() {
	case "esc":
		return true, m, func() tea.Msg { return FilterCancelledMsg{} }

	case "enter":
		return m.applyFilter()

	case "tab", "down":
		m.inputs[m.activeField].Blur()
		m.activeField = (m.activeField + 1) % filterFieldCount
		return true, m, m.inputs[m.activeField].Focus()

	case "shift+tab", "up":
		m.inputs[m.activeField].Blur()
		m.activeField = (m.activeField - 1 + filterFieldCount) % filterFieldCount
		return true, m, m.inputs[m.activeField].Focus()

	case "ctrl+l":
		for i := range m.inputs {
			m.inputs[i].Reset()
		}
		m.severityIdx = 0
		m.timeRangeIdx = 0
		m.bookmarkedIdx = 0
		m.liveOnlyIdx = 0
		m.errMsg = ""
		return true, m, nil
	}

	if m.handleOptionCycle(msg.String()) {
		return true, m, nil
	}
	return false, m, nil
}

// applyFilter validates the current inputs and either emits FilterAppliedMsg or
// keeps the overlay open with an inline error message.
func (m FilterModel) applyFilter() (handled bool, model FilterModel, cmd tea.Cmd) {
	f, err := m.buildFilter()
	if err != nil {
		m.errMsg = err.Error()
		return true, m, nil
	}
	m.errMsg = ""
	return true, m, func() tea.Msg { return FilterAppliedMsg{Filter: f} }
}

// handleOptionCycle advances the focused option field (severity, time range,
// bookmarked, live-only) on left/right/h/l, returning true when it cycled one.
func (m *FilterModel) handleOptionCycle(key string) bool {
	var delta int
	switch key {
	case "left", "h":
		delta = -1
	case "right", "l":
		delta = 1
	default:
		return false
	}

	switch m.activeField {
	case filterFieldSeverity:
		m.severityIdx = wrapIndex(m.severityIdx, delta, len(severityOptions))
		m.inputs[filterFieldSeverity].SetValue(severityOptions[m.severityIdx])
	case filterFieldTimeRange:
		m.timeRangeIdx = wrapIndex(m.timeRangeIdx, delta, len(timeRangeOptions))
		m.inputs[filterFieldTimeRange].SetValue(timeRangeOptions[m.timeRangeIdx])
	case filterFieldBookmarked:
		m.bookmarkedIdx = wrapIndex(m.bookmarkedIdx, delta, len(bookmarkedOptions))
		m.inputs[filterFieldBookmarked].SetValue(bookmarkedOptions[m.bookmarkedIdx])
	case filterFieldLiveOnly:
		m.liveOnlyIdx = wrapIndex(m.liveOnlyIdx, delta, len(liveOnlyOptions))
		m.inputs[filterFieldLiveOnly].SetValue(liveOnlyOptions[m.liveOnlyIdx])
	default:
		return false
	}
	return true
}

// wrapIndex advances idx by delta within [0, n), wrapping around either end.
func wrapIndex(idx, delta, n int) int {
	return (idx + delta + n) % n
}

func (m FilterModel) View() string {
	title := StyleTitle.Render("Filter Hits")

	var fields strings.Builder
	for _, input := range m.inputs {
		fields.WriteString(input.View() + "\n")
	}

	help := StyleHelpKey.Render("enter") + StyleHelpDesc.Render(" apply") + "  " +
		StyleHelpKey.Render("esc") + StyleHelpDesc.Render(" cancel") + "  " +
		StyleHelpKey.Render("tab") + StyleHelpDesc.Render(" next field") + "  " +
		StyleHelpKey.Render("◄/►") + StyleHelpDesc.Render(" cycle") + "  " +
		StyleHelpKey.Render("ctrl+l") + StyleHelpDesc.Render(" clear all")

	sections := []string{title, "", fields.String()}
	if m.errMsg != "" {
		sections = append(sections, StyleHighSeverity.Render("⚠ "+m.errMsg))
	}
	sections = append(sections, help)
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

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

// buildFilter translates the overlay inputs into a QueryFilter, returning an
// error for the first invalid field rather than a filter that matches nothing.
func (m FilterModel) buildFilter() (domain.QueryFilter, error) {
	var f domain.QueryFilter

	f.Keyword = m.inputs[filterFieldKeyword].Value()
	f.Session = m.inputs[filterFieldSession].Value()
	f.TLD = m.inputs[filterFieldTLD].Value()

	if err := m.applyScoreMin(&f); err != nil {
		return domain.QueryFilter{}, err
	}
	if err := m.applySeverity(&f); err != nil {
		return domain.QueryFilter{}, err
	}
	if err := m.applyTimeRange(&f); err != nil {
		return domain.QueryFilter{}, err
	}
	m.applyBookmarked(&f)
	m.applyLiveOnly(&f)

	return f, nil
}

// applyScoreMin parses and validates the min-score field.
func (m FilterModel) applyScoreMin(f *domain.QueryFilter) error {
	v := strings.TrimSpace(m.inputs[filterFieldScoreMin].Value())
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fmt.Errorf("min score must be a non-negative number, got %q", v)
	}
	f.ScoreMin = n
	return nil
}

// applySeverity validates the severity field against the allowed values.
func (m FilterModel) applySeverity(f *domain.QueryFilter) error {
	v := strings.TrimSpace(m.inputs[filterFieldSeverity].Value())
	if v == "" {
		return nil
	}
	for _, opt := range severityOptions {
		if opt != "" && opt == v {
			f.Severity = v
			return nil
		}
	}
	return fmt.Errorf("severity must be HIGH, MED, or LOW, got %q", v)
}

// applyTimeRange validates the time-range field against the allowed values.
func (m FilterModel) applyTimeRange(f *domain.QueryFilter) error {
	v := strings.TrimSpace(m.inputs[filterFieldTimeRange].Value())
	if v == "" {
		return nil
	}
	d := parseTimeRange(v)
	if d == 0 {
		return fmt.Errorf("time range must be one of 1h, 6h, 12h, 24h, 7d, got %q", v)
	}
	f.Since = d
	return nil
}

// applyBookmarked maps the tri-state bookmarked field onto the filter.
func (m FilterModel) applyBookmarked(f *domain.QueryFilter) {
	switch m.inputs[filterFieldBookmarked].Value() {
	case "yes":
		yes := true
		f.Bookmarked = &yes
	case "no":
		no := false
		f.Bookmarked = &no
	}
}

// applyLiveOnly maps the live-only field onto the filter.
func (m FilterModel) applyLiveOnly(f *domain.QueryFilter) {
	if m.inputs[filterFieldLiveOnly].Value() == "yes" {
		f.LiveOnly = true
	}
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
