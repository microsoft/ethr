package payloads

import (
	"sort"
	"time"

	"weavelab.xyz/ethr/session"

	"weavelab.xyz/ethr/ethr"
)

type LatencyPayload struct {
	RemoteIP string
	Protocol ethr.Protocol
	Raw      []time.Duration
	Avg      time.Duration
	Min      time.Duration
	Max      time.Duration
	P50      time.Duration
	P90      time.Duration
	P95      time.Duration
	P99      time.Duration
	P999     time.Duration
	P9999    time.Duration
}

func NewLatencies(test *session.Test, rttCount int, latencies []time.Duration) LatencyPayload {
	sum := int64(0)
	for _, d := range latencies {
		sum += d.Nanoseconds()
	}
	elapsed := time.Duration(sum / int64(rttCount))
	sort.SliceStable(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	//
	// Special handling for rttCount == 1. This prevents negative index
	// in the latencyNumber index. The other option is to use
	// roundUpToZero() but that is more expensive.
	//
	rttCountFixed := rttCount
	if rttCountFixed == 1 {
		rttCountFixed = 2
	}
	return LatencyPayload{
		RemoteIP: test.RemoteIP,
		Protocol: test.ID.Protocol,
		Raw:      latencies,
		Avg:      elapsed,
		Min:      latencies[0],
		Max:      latencies[rttCount-1],
		P50:      latencies[((rttCountFixed*50)/100)-1],
		P90:      latencies[((rttCountFixed*90)/100)-1],
		P95:      latencies[((rttCountFixed*95)/100)-1],
		P99:      latencies[((rttCountFixed*99)/100)-1],
		P999:     latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
		P9999:    latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
	}
}
