package tcp

import (
	"fmt"
	"io"
	"net"
	"sort"
	"sync/atomic"
	"time"
	"weavelab.xyz/ethr/ethr"
)

func (h Handler) TestLatency(test *ethr.Test, clientParam ethr.ClientParams, conn net.Conn) (*ethr.LatencyResult, error) {
	bytes := make([]byte, clientParam.BufferSize)
	rttCount := clientParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	for {
		_, err := io.ReadFull(conn, bytes)
		if err != nil {
			return nil, fmt.Errorf("error receiving data for latency tests: %w", err)
		}
		for i := uint32(0); i < rttCount; i++ {
			s1 := time.Now()
			_, err = conn.Write(bytes)
			if err != nil {
				return nil, fmt.Errorf("error sending data for latency test: %w", err)

			}
			_, err = io.ReadFull(conn, bytes)
			if err != nil {
				return nil, fmt.Errorf("error receiving data for latency test: %w", err)

			}
			e2 := time.Since(s1)
			latencyNumbers[i] = e2
		}
		sum := int64(0)
		for _, d := range latencyNumbers {
			sum += d.Nanoseconds()
		}
		elapsed := time.Duration(sum / int64(rttCount))
		sort.SliceStable(latencyNumbers, func(i, j int) bool {
			return latencyNumbers[i] < latencyNumbers[j]
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
		atomic.SwapUint64(&test.Result.Latency, uint64(elapsed.Nanoseconds()))
		return  &ethr.LatencyResult{
			RemoteIP: test.RemoteIP,
			Protocol: test.ID.Protocol,
			Avg:      elapsed,
			Min:      latencyNumbers[0],
			Max:      latencyNumbers[rttCount-1],
			P50:      latencyNumbers[((rttCountFixed*50)/100)-1],
			P90:      latencyNumbers[((rttCountFixed*90)/100)-1],
			P95:      latencyNumbers[((rttCountFixed*95)/100)-1],
			P99:      latencyNumbers[((rttCountFixed*99)/100)-1],
			P999:     latencyNumbers[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
			P9999:    latencyNumbers[uint64(((float64(rttCountFixed)*99.99)/100)-1)],
		}, nil
	}
}
