package payloads

import (
	"fmt"
	"math"
	"sort"
	"time"

	"weavelab.xyz/ethr/ui"
)

type RawLatencies struct {
	Latencies []time.Duration
}

type LatencyPayload struct {
	Raw    []time.Duration
	Jitter time.Duration
	Avg    time.Duration
	Min    time.Duration
	Max    time.Duration
	P50    time.Duration
	P90    time.Duration
	P95    time.Duration
	P99    time.Duration
	P999   time.Duration
	P9999  time.Duration
}

func (p LatencyPayload) String() string {
	return fmt.Sprintf("%9s %9s %9s %9s %9s %9s %9s %9s %9s %9s",
		ui.DurationToString(p.Avg),
		ui.DurationToString(p.Min),
		ui.DurationToString(p.P50),
		ui.DurationToString(p.P90),
		ui.DurationToString(p.P95),
		ui.DurationToString(p.P99),
		ui.DurationToString(p.P999),
		ui.DurationToString(p.P9999),
		ui.DurationToString(p.Max),
		ui.DurationToString(p.Jitter))
}

func NewLatencies(latencies []time.Duration) LatencyPayload {
	rttCount := len(latencies)
	if rttCount == 0 {
		return LatencyPayload{}
	}

	//
	// Special handling for rttCount == 1. This prevents negative index
	// in the latencyNumber index. The other option is to use
	// roundUpToZero() but that is more expensive.
	//
	rttCountFixed := rttCount
	if rttCountFixed == 1 {
		rttCountFixed = 2
	}

	sum := int64(0)
	diffs := int64(0)
	for i := 0; i < len(latencies); i++ {
		sum += latencies[i].Nanoseconds()
		if i > 0 {
			diffs += int64(math.Abs(float64(latencies[i].Nanoseconds() - latencies[i-1].Nanoseconds())))
		}
	}

	elapsed := time.Duration(sum / int64(rttCount))
	jitter := time.Duration(diffs/int64(rttCountFixed) - 1)
	sort.SliceStable(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	return LatencyPayload{
		Raw:    latencies,
		Jitter: jitter,
		Avg:    elapsed,
		Min:    latencies[0],
		Max:    latencies[rttCount-1],
		P50:    latencies[((rttCountFixed*50)/100)-1],
		P90:    latencies[((rttCountFixed*90)/100)-1],
		P95:    latencies[((rttCountFixed*95)/100)-1],
		P99:    latencies[((rttCountFixed*99)/100)-1],
		P999:   latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
		P9999:  latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
	}
}
