package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// minClusterSize is the smallest number of distinct domains that makes a shared
// IP interesting. A single domain on an IP is not a cluster; two or more sharing
// one address is the weakest signal of common infrastructure worth surfacing.
const minClusterSize = 2

// clusterSampleLimit caps how many member domains are previewed per cluster.
const clusterSampleLimit = 5

// cdnEdgeProviders are providers whose addresses are shared CDN edge nodes.
// Co-hosting on these carries no clustering signal — thousands of unrelated
// sites sit behind the same Cloudflare/Fastly/Akamai edge IP — so clusters whose
// detected provider is one of these are excluded. Dedicated cloud hosts
// (aws/gcp/azure/digitalocean) are NOT excluded: N flagged domains on one of
// those is far more likely to be a single operator.
var cdnEdgeProviders = map[string]bool{
	"cloudflare": true,
	"fastly":     true,
	"akamai":     true,
}

// isCDNEdge reports whether the detected hosting provider is shared CDN edge
// infrastructure. Matching is case-insensitive and substring-based so values
// such as "Cloudflare, Inc." still match.
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

// NetworkClusters groups stored hits by shared resolved IP and returns one
// aggregate per IP that hosts two or more distinct domains. Each row carries the
// domain/live counts, detected provider, peak score, certificate time span, and
// a bounded sample of the highest-scoring member domains. CDN-edge clusters are
// excluded — co-hosting behind a shared edge is meaningless for attribution.
//
// The query uses json_each over resolved_ips to expand each hit's IP array into
// one row per (domain, IP) pair, then aggregates by IP. All values are produced
// by SQLite aggregates; nothing user-controlled is interpolated.
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

// clusterSampleDomains returns up to clusterSampleLimit member domains for a
// shared IP, highest score first, for preview in the cluster row. The IP is
// bound as a parameter.
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
	// Order (score DESC, domain ASC) is fixed by the query — highest-scoring
	// members preview first.
	return domains, nil
}
