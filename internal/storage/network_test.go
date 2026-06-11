package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// enrichedHit returns a hit with enrichment fields set, for clustering tests.
func enrichedHit(domainName string, score int, ips []string, provider string, live bool) domain.Hit {
	h := testHit(domainName, score, domain.SeverityHigh)
	h.ResolvedIPs = ips
	h.HostingProvider = provider
	h.IsLive = live
	return h
}

// seedHits upserts each hit into the store, failing the test on error.
func seedHits(t *testing.T, db *DB, hits ...domain.Hit) {
	t.Helper()
	ctx := context.Background()
	for _, h := range hits {
		require.NoError(t, db.UpsertHit(ctx, h))
	}
}

// clusterByIP finds the cluster for an IP, or fails the test.
func clusterByIP(t *testing.T, clusters []domain.NetworkCluster, ip string) domain.NetworkCluster {
	t.Helper()
	for _, c := range clusters {
		if c.IP == ip {
			return c
		}
	}
	t.Fatalf("no cluster found for IP %s", ip)
	return domain.NetworkCluster{}
}

func TestNetworkClusters(t *testing.T) {
	tests := []struct {
		name   string
		hits   []domain.Hit
		assert func(t *testing.T, clusters []domain.NetworkCluster)
	}{
		{
			name: "two domains on a shared dedicated IP form a cluster",
			hits: []domain.Hit{
				enrichedHit("scam-one.xyz", 9, []string{"5.5.5.5"}, "digitalocean", true),
				enrichedHit("scam-two.xyz", 7, []string{"5.5.5.5"}, "digitalocean", false),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				require.Len(t, clusters, 1)
				c := clusters[0]
				assert.Equal(t, "5.5.5.5", c.IP)
				assert.Equal(t, 2, c.DomainCount)
				assert.Equal(t, 1, c.LiveCount)
				assert.Equal(t, 9, c.MaxScore)
				assert.Equal(t, "digitalocean", c.HostingProvider)
				assert.ElementsMatch(t, []string{"scam-one.xyz", "scam-two.xyz"}, c.SampleDomains)
			},
		},
		{
			name: "single domain on an IP is below the threshold",
			hits: []domain.Hit{
				enrichedHit("lonely.xyz", 9, []string{"6.6.6.6"}, "digitalocean", true),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				assert.Empty(t, clusters)
			},
		},
		{
			name: "cloudflare-hosted shared IP is excluded as CDN noise",
			hits: []domain.Hit{
				enrichedHit("cdn-a.com", 9, []string{"104.16.1.1"}, "cloudflare", true),
				enrichedHit("cdn-b.com", 8, []string{"104.16.1.1"}, "cloudflare", true),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				assert.Empty(t, clusters)
			},
		},
		{
			name: "fastly and akamai edges are also excluded",
			hits: []domain.Hit{
				enrichedHit("f-a.com", 9, []string{"151.101.1.1"}, "fastly", true),
				enrichedHit("f-b.com", 8, []string{"151.101.1.1"}, "fastly", true),
				enrichedHit("a-a.com", 9, []string{"23.1.1.1"}, "akamai", true),
				enrichedHit("a-b.com", 8, []string{"23.1.1.1"}, "akamai", true),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				assert.Empty(t, clusters)
			},
		},
		{
			name: "clusters are sorted by domain count descending",
			hits: []domain.Hit{
				enrichedHit("small-a.xyz", 5, []string{"7.7.7.7"}, "aws", false),
				enrichedHit("small-b.xyz", 5, []string{"7.7.7.7"}, "aws", false),
				enrichedHit("big-a.xyz", 5, []string{"8.8.8.8"}, "aws", false),
				enrichedHit("big-b.xyz", 5, []string{"8.8.8.8"}, "aws", false),
				enrichedHit("big-c.xyz", 5, []string{"8.8.8.8"}, "aws", false),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				require.Len(t, clusters, 2)
				assert.Equal(t, "8.8.8.8", clusters[0].IP)
				assert.Equal(t, 3, clusters[0].DomainCount)
				assert.Equal(t, "7.7.7.7", clusters[1].IP)
				assert.Equal(t, 2, clusters[1].DomainCount)
			},
		},
		{
			name: "a domain with multiple IPs counts toward each shared IP",
			hits: []domain.Hit{
				enrichedHit("multi.xyz", 9, []string{"9.9.9.9", "10.10.10.10"}, "aws", true),
				enrichedHit("peer-a.xyz", 6, []string{"9.9.9.9"}, "aws", false),
				enrichedHit("peer-b.xyz", 6, []string{"10.10.10.10"}, "aws", false),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				require.Len(t, clusters, 2)
				a := clusterByIP(t, clusters, "9.9.9.9")
				b := clusterByIP(t, clusters, "10.10.10.10")
				assert.Equal(t, 2, a.DomainCount)
				assert.Equal(t, 2, b.DomainCount)
				assert.Contains(t, a.SampleDomains, "multi.xyz")
				assert.Contains(t, b.SampleDomains, "multi.xyz")
			},
		},
		{
			name: "empty resolved_ips never produce a cluster",
			hits: []domain.Hit{
				enrichedHit("none-a.xyz", 9, nil, "", false),
				enrichedHit("none-b.xyz", 9, []string{}, "", false),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				assert.Empty(t, clusters)
			},
		},
		{
			name: "sample domains are capped at the preview limit",
			hits: []domain.Hit{
				enrichedHit("d1.xyz", 9, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d2.xyz", 8, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d3.xyz", 7, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d4.xyz", 6, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d5.xyz", 5, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d6.xyz", 4, []string{"11.11.11.11"}, "aws", false),
				enrichedHit("d7.xyz", 3, []string{"11.11.11.11"}, "aws", false),
			},
			assert: func(t *testing.T, clusters []domain.NetworkCluster) {
				require.Len(t, clusters, 1)
				assert.Equal(t, 7, clusters[0].DomainCount)
				assert.Len(t, clusters[0].SampleDomains, clusterSampleLimit)
				// Highest-scoring members preview first.
				assert.Equal(t, "d1.xyz", clusters[0].SampleDomains[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			seedHits(t, db, tt.hits...)
			clusters, err := db.NetworkClusters(context.Background())
			require.NoError(t, err)
			tt.assert(t, clusters)
		})
	}
}

func TestNetworkClusters_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	clusters, err := db.NetworkClusters(context.Background())
	require.NoError(t, err)
	assert.Empty(t, clusters)
}

func TestQueryHits_SharedIPFilter(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	seedHits(t, db,
		enrichedHit("a.xyz", 9, []string{"1.2.3.4", "5.6.7.8"}, "aws", true),
		enrichedHit("b.xyz", 7, []string{"1.2.3.4"}, "aws", false),
		enrichedHit("c.xyz", 5, []string{"9.9.9.9"}, "aws", false),
	)

	tests := []struct {
		name string
		ip   string
		want []string
	}{
		{"shared IP returns both members", "1.2.3.4", []string{"a.xyz", "b.xyz"}},
		{"secondary IP of a multi-IP domain matches", "5.6.7.8", []string{"a.xyz"}},
		{"unrelated IP returns its single member", "9.9.9.9", []string{"c.xyz"}},
		{"unknown IP returns nothing", "0.0.0.0", nil},
		{"partial IP does not spuriously match", "1.2.3", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits, err := db.QueryHits(ctx, domain.QueryFilter{SharedIP: tt.ip, Limit: 50})
			require.NoError(t, err)
			got := make([]string, 0, len(hits))
			for _, h := range hits {
				got = append(got, h.Domain)
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestQueryHits_SharedIPRoundTripWithClusters(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	seedHits(t, db,
		enrichedHit("op-1.xyz", 9, []string{"203.0.113.7"}, "digitalocean", true),
		enrichedHit("op-2.xyz", 8, []string{"203.0.113.7"}, "digitalocean", true),
		enrichedHit("op-3.xyz", 7, []string{"203.0.113.7"}, "digitalocean", false),
	)

	clusters, err := db.NetworkClusters(ctx)
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	require.Equal(t, 3, clusters[0].DomainCount)

	// Drilling into the cluster's IP must return exactly its members.
	hits, err := db.QueryHits(ctx, domain.QueryFilter{SharedIP: clusters[0].IP, Limit: 50})
	require.NoError(t, err)
	require.Len(t, hits, clusters[0].DomainCount)
}
