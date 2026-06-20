package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// minClusterSize is the smallest distinct-domain count worth surfacing as a
// shared-IP cluster; one domain on an IP is not a cluster.
const minClusterSize = 2

const clusterSampleLimit = 5

// cdnEdgeProviders host thousands of unrelated sites per edge IP, so co-hosting
// is no signal and these clusters are dropped; dedicated clouds are not excluded.
var cdnEdgeProviders = map[string]bool{
	"cloudflare": true,
	"fastly":     true,
	"akamai":     true,
}

// isCDNEdge matches case-insensitively by substring so "Cloudflare, Inc." hits.
func isCDNEdge(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return false
	}
	for edge := range cdnEdgeProviders {
		if strings.Contains(p, edge) {
			return true
		}
	}
	return false
}

// NetworkClusters returns one aggregate per IP hosting two or more distinct
// domains, excluding CDN-edge IPs whose co-hosting is meaningless for attribution.
func (d *DB) NetworkClusters(ctx context.Context) ([]domain.NetworkCluster, error) {
	const query = `
		SELECT
			ip,
			COUNT(DISTINCT domain) AS domain_count,
			SUM(is_live_max) AS live_count,
			MAX(score) AS max_score,
			MIN(cert_not_before) AS first_seen,
			MAX(cert_not_before) AS last_seen,
			COALESCE(
				(SELECT hosting_provider FROM (
					SELECT h2.hosting_provider, COUNT(*) AS c
					FROM hits h2, json_each(h2.resolved_ips) je2
					WHERE je2.value = m.ip AND h2.hosting_provider != ''
					GROUP BY h2.hosting_provider
					ORDER BY c DESC
					LIMIT 1
				)),
				''
			) AS provider
		FROM (
			SELECT
				je.value AS ip,
				h.domain AS domain,
				h.score AS score,
				h.cert_not_before AS cert_not_before,
				MAX(h.is_live) AS is_live_max
			FROM hits h, json_each(h.resolved_ips) je
			WHERE je.value != ''
			GROUP BY je.value, h.domain
		) m
		GROUP BY ip
		HAVING COUNT(DISTINCT domain) >= ?
		ORDER BY domain_count DESC, max_score DESC, ip ASC`

	rows, err := d.db.QueryContext(ctx, query, minClusterSize)
	if err != nil {
		return nil, fmt.Errorf("querying network clusters: %w", err)
	}
	defer rows.Close()

	var clusters []domain.NetworkCluster
	for rows.Next() {
		var (
			c         domain.NetworkCluster
			firstStr  string
			lastStr   string
			liveCount int
		)
		if err := rows.Scan(&c.IP, &c.DomainCount, &liveCount, &c.MaxScore, &firstStr, &lastStr, &c.HostingProvider); err != nil {
			return nil, fmt.Errorf("scanning network cluster: %w", err)
		}
		// Drop CDN-edge clusters: co-hosting on shared edge IPs is not a signal.
		if isCDNEdge(c.HostingProvider) {
			continue
		}
		c.LiveCount = liveCount
		c.FirstSeen = parseTimestamp(firstStr)
		c.LastSeen = parseTimestamp(lastStr)

		sample, err := d.clusterSampleDomains(ctx, c.IP)
		if err != nil {
			return nil, err
		}
		c.SampleDomains = sample

		clusters = append(clusters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating network cluster rows: %w", err)
	}

	return clusters, nil
}

// clusterSampleDomains returns up to clusterSampleLimit member domains for an
// IP, highest score first, for the cluster-row preview.
func (d *DB) clusterSampleDomains(ctx context.Context, ip string) ([]string, error) {
	const query = `
		SELECT DISTINCT domain, score
		FROM hits h, json_each(h.resolved_ips) je
		WHERE je.value = ?
		ORDER BY score DESC, domain ASC
		LIMIT ?`

	rows, err := d.db.QueryContext(ctx, query, ip, clusterSampleLimit)
	if err != nil {
		return nil, fmt.Errorf("querying cluster sample for %s: %w", ip, err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var (
			dom   string
			score int
		)
		if err := rows.Scan(&dom, &score); err != nil {
			return nil, fmt.Errorf("scanning cluster sample row: %w", err)
		}
		domains = append(domains, dom)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cluster sample rows: %w", err)
	}
	return domains, nil
}
