package domain

import "time"

// NetworkCluster is an aggregate view of every domain that resolves to a shared
// IP address. It is the headline output of the network-clustering feature: a
// single dedicated host serving N distinct flagged domains is one operator's
// infrastructure, and grouping on the shared IP surfaces the whole campaign at
// once. CDN-edge addresses (where co-hosting is meaningless) are excluded by the
// store before clustering.
type NetworkCluster struct {
	// IP is the shared resolved IP address that defines the cluster.
	IP string

	// DomainCount is the number of distinct flagged domains resolving to IP.
	DomainCount int

	// LiveCount is how many of those domains responded to the liveness probe.
	LiveCount int

	// HostingProvider is the detected provider for the cluster, or empty/"unknown"
	// when undetected. CDN-edge providers are excluded upstream, so a populated
	// value here is a dedicated host rather than shared edge infrastructure.
	HostingProvider string

	// MaxScore is the highest hit score among the cluster's domains — a quick
	// read on how threatening the most severe member is.
	MaxScore int

	// FirstSeen is the earliest certificate timestamp across the cluster.
	FirstSeen time.Time

	// LastSeen is the latest certificate timestamp across the cluster.
	LastSeen time.Time

	// SampleDomains holds a bounded preview of member domains (highest score
	// first) for display without loading the full membership.
	SampleDomains []string
}
