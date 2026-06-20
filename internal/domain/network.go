package domain

import "time"

// NetworkCluster aggregates every flagged domain resolving to one shared IP,
// surfacing an operator's whole campaign; CDN-edge IPs are excluded upstream.
type NetworkCluster struct {
	IP string

	DomainCount int

	LiveCount int

	// CDN-edge providers are excluded upstream, so a value here is a dedicated host.
	HostingProvider string

	MaxScore int

	FirstSeen time.Time

	LastSeen time.Time

	// Bounded preview (highest score first) for display without loading full membership.
	SampleDomains []string
}
