// Package enrichment probes discovered domains for DNS, hosting provider, and
// HTTP liveness, persisting results and publishing them for the TUI.
package enrichment

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// EnrichResult carries the outcome of a single domain enrichment probe.
type EnrichResult struct {
	Domain          string
	IsLive          bool
	ResolvedIPs     []string
	HostingProvider string
	HTTPStatus      int
	Error           error
}

const maxWorkers = 5

const queueCapacity = 1000

// Enricher runs a rate-limited worker pool probing domains for DNS and HTTP
// liveness, persisting results and publishing them downstream.
type Enricher struct {
	store      domain.Store
	httpClient *http.Client
	limiter    *rate.Limiter
	resultCh   chan<- EnrichResult
	queue      chan string
	wg         sync.WaitGroup
}

// NewEnricher builds an Enricher; the caller must start the loop via Run.
func NewEnricher(store domain.Store, enrichCh chan<- EnrichResult) *Enricher {
	return &Enricher{
		store: store,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		// Global limit shared across all workers: 5 req/sec, burst maxWorkers.
		limiter:  rate.NewLimiter(rate.Limit(5), maxWorkers),
		resultCh: enrichCh,
		queue:    make(chan string, queueCapacity),
	}
}

// Enqueue adds a domain to the enrichment probe queue. If the queue is
// full the domain is silently dropped to avoid blocking the poller.
func (e *Enricher) Enqueue(domainName string) {
	select {
	case e.queue <- domainName:
	default:
		slog.Warn("enrichment queue full, dropping domain", "domain", domainName)
	}
}

// Run starts the worker pool and blocks until ctx is cancelled and all
// workers have drained.
func (e *Enricher) Run(ctx context.Context) error {
	for i := 0; i < maxWorkers; i++ {
		e.wg.Add(1)
		go e.worker(ctx)
	}

	<-ctx.Done()
	e.wg.Wait()
	return ctx.Err()
}

// worker is a single enrichment goroutine that dequeues domains, waits
// for rate limiter tokens, runs DNS + HTTP probes, and publishes results.
func (e *Enricher) worker(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case domainName, ok := <-e.queue:
			if !ok {
				return
			}
			if err := e.limiter.Wait(ctx); err != nil {
				return // context cancelled
			}
			e.probe(ctx, domainName)
		}
	}
}

// probe runs DNS resolution and HTTP liveness check for a single domain,
// persists the enrichment data, and sends the result downstream.
func (e *Enricher) probe(ctx context.Context, domainName string) {
	result := EnrichResult{Domain: domainName}

	ips, provider, err := ResolveDomain(ctx, domainName)
	if err != nil {
		slog.Debug("DNS resolution failed", "domain", domainName, "error", err)
	}
	result.ResolvedIPs = ips
	result.HostingProvider = provider

	statusCode, isLive, err := ProbeLiveness(ctx, e.httpClient, domainName)
	if err != nil {
		slog.Debug("HTTP probe failed", "domain", domainName, "error", err)
	}
	result.IsLive = isLive
	result.HTTPStatus = statusCode

	if err := e.store.UpdateEnrichment(ctx, domainName, result.IsLive, result.ResolvedIPs, result.HostingProvider, result.HTTPStatus); err != nil {
		result.Error = err
		slog.Warn("failed to persist enrichment", "domain", domainName, "error", err)
	}

	select {
	case e.resultCh <- result:
	case <-ctx.Done():
	}
}
