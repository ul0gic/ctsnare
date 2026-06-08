package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/poller"
)

func TestAggregatePollStats_SumsAndDerivesRates(t *testing.T) {
	perLog := map[string]poller.PollStats{
		"Argon": {LogName: "Argon", CertsScanned: 600, HitsFound: 6},
		"Xenon": {LogName: "Xenon", CertsScanned: 600, HitsFound: 6},
	}

	// Over two minutes: 1200 certs / 120s = 10 c/s; 12 hits / 2 min = 6 hits/min.
	agg := aggregatePollStats(perLog, 2*time.Minute)

	assert.Equal(t, int64(1200), agg.CertsScanned)
	assert.Equal(t, int64(12), agg.HitsFound)
	assert.Equal(t, 2, agg.ActiveLogs)
	assert.InDelta(t, 10.0, agg.CertsPerSec, 1e-9, "certs/sec = total/elapsedSeconds")
	assert.InDelta(t, 6.0, agg.HitsPerMin, 1e-9, "hits/min = total/elapsedMinutes")
}

func TestAggregatePollStats_HitsPerMinPositiveAfterHits(t *testing.T) {
	// BUG-002 regression: HitsPerMin must be live, not a hardcoded zero.
	perLog := map[string]poller.PollStats{
		"Argon": {LogName: "Argon", CertsScanned: 100, HitsFound: 3},
	}
	agg := aggregatePollStats(perLog, 30*time.Second)
	assert.Positive(t, agg.HitsPerMin, "HitsPerMin must be > 0 once hits accumulate")
	// 3 hits / 0.5 min = 6 hits/min.
	assert.InDelta(t, 6.0, agg.HitsPerMin, 1e-9)
}

func TestAggregatePollStats_ZeroElapsed_NoDivideByZero(t *testing.T) {
	perLog := map[string]poller.PollStats{
		"Argon": {LogName: "Argon", CertsScanned: 100, HitsFound: 3},
	}
	agg := aggregatePollStats(perLog, 0)
	assert.Zero(t, agg.CertsPerSec)
	assert.Zero(t, agg.HitsPerMin)
}
