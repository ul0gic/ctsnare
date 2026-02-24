package tui

import "github.com/ul0gic/ctsnare/internal/domain"

// HitMsg is sent when a new scored domain hit arrives from the poller.
type HitMsg struct {
	Hit domain.Hit
}

// StatsMsg is sent when polling statistics are updated.
type StatsMsg struct {
	Stats PollStats
}

// HitsLoadedMsg is sent when a batch of hits has been loaded from the database.
type HitsLoadedMsg struct {
	Hits []domain.Hit
}

// SwitchViewMsg requests a switch to a different view.
type SwitchViewMsg struct {
	View int
}

// ShowDetailMsg requests showing the detail view for a specific hit.
type ShowDetailMsg struct {
	Hit domain.Hit
}

// PollStats contains aggregate statistics from all pollers for TUI display.
type PollStats struct {
	CertsScanned int64
	HitsFound    int64
	CertsPerSec  float64
	ActiveLogs   int
}
