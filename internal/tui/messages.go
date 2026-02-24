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
// It is computed by bridging and summing per-log poller.PollStats values.
type PollStats struct {
	// CertsScanned is the total number of certificate entries processed across all logs.
	CertsScanned int64

	// HitsFound is the total number of domains that scored above zero and were stored.
	HitsFound int64

	// CertsPerSec is the average processing rate since the watch command started.
	CertsPerSec float64

	// ActiveLogs is the number of CT log pollers currently running.
	ActiveLogs int

	// HitsPerMin is the rolling rate of new hits stored per minute.
	HitsPerMin float64
}

// EnrichmentMsg is sent when the enrichment pipeline completes a liveness probe for a domain.
type EnrichmentMsg struct {
	Domain          string
	IsLive          bool
	ResolvedIPs     []string
	HostingProvider string
	HTTPStatus      int
}

// BookmarkToggleMsg is sent when a hit's bookmark state has been toggled.
type BookmarkToggleMsg struct {
	Domain     string
	Bookmarked bool
}

// DeleteHitsMsg is sent when one or more hits have been deleted from storage.
type DeleteHitsMsg struct {
	Domains []string
}

// DiscardedDomainMsg is sent when a domain scored zero and was discarded.
// Used for the activity feed to show scanning activity even when no hits are produced.
type DiscardedDomainMsg struct {
	Domain string
}
