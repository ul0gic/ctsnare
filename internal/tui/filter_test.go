package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setField sets the value of a filter input field by index.
func setField(m *FilterModel, field int, value string) {
	m.inputs[field].SetValue(value)
}

func TestBuildFilter_TLDAndLiveOnly(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldTLD, "xyz")
	setField(&m, filterFieldLiveOnly, "yes")

	f, err := m.buildFilter()
	require.NoError(t, err)
	assert.Equal(t, "xyz", f.TLD, "TLD field must thread into the filter")
	assert.True(t, f.LiveOnly, "live-only yes must set LiveOnly")
}

func TestBuildFilter_LiveOnlyDefaultOff(t *testing.T) {
	m := NewFilterModel()
	f, err := m.buildFilter()
	require.NoError(t, err)
	assert.False(t, f.LiveOnly)
	assert.Empty(t, f.TLD)
}

func TestBuildFilter_BookmarkedTriState(t *testing.T) {
	tests := []struct {
		value string
		want  *bool
	}{
		{"", nil},
		{"yes", boolPtrT(true)},
		{"no", boolPtrT(false)},
	}
	for _, tt := range tests {
		t.Run("bookmarked="+tt.value, func(t *testing.T) {
			m := NewFilterModel()
			setField(&m, filterFieldBookmarked, tt.value)
			f, err := m.buildFilter()
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, f.Bookmarked)
				return
			}
			require.NotNil(t, f.Bookmarked, "non-empty bookmarked must set a pointer")
			assert.Equal(t, *tt.want, *f.Bookmarked)
		})
	}
}

func TestBuildFilter_ValidScoreSeverityTimeRange(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldScoreMin, "7")
	setField(&m, filterFieldSeverity, "HIGH")
	setField(&m, filterFieldTimeRange, "24h")

	f, err := m.buildFilter()
	require.NoError(t, err)
	assert.Equal(t, 7, f.ScoreMin)
	assert.Equal(t, "HIGH", f.Severity)
	assert.Equal(t, 24*time.Hour, f.Since)
}

func TestBuildFilter_RejectsInvalidScore(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldScoreMin, "abc")
	_, err := m.buildFilter()
	require.Error(t, err, "non-numeric score must be rejected, not silently dropped")
	assert.Contains(t, err.Error(), "min score")
}

func TestBuildFilter_RejectsInvalidSeverity(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldSeverity, "MEDIUM") // not a valid severity token
	_, err := m.buildFilter()
	require.Error(t, err, "invalid severity must be rejected")
	assert.Contains(t, err.Error(), "severity")
}

func TestBuildFilter_RejectsInvalidTimeRange(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldTimeRange, "30m") // not in the option list
	_, err := m.buildFilter()
	require.Error(t, err, "invalid time range must be rejected")
	assert.Contains(t, err.Error(), "time range")
}

func TestFilterApply_InvalidInputKeepsOverlayOpenWithError(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldScoreMin, "abc")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// No FilterAppliedMsg should be emitted; the overlay stays open with an error.
	if cmd != nil {
		msg := cmd()
		_, applied := msg.(FilterAppliedMsg)
		assert.False(t, applied, "invalid apply must not emit FilterAppliedMsg")
	}
	assert.NotEmpty(t, updated.errMsg, "invalid apply must surface an inline error")
}

func TestFilterApply_ValidInputEmitsAppliedMsg(t *testing.T) {
	m := NewFilterModel()
	setField(&m, filterFieldSeverity, "MED")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	applied, ok := msg.(FilterAppliedMsg)
	require.True(t, ok, "valid apply must emit FilterAppliedMsg, got %T", msg)
	assert.Equal(t, "MED", applied.Filter.Severity)
}

func boolPtrT(b bool) *bool { return &b }
