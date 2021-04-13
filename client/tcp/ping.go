package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"weavelab.xyz/ethr/client/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestPing(test *session.Test, g time.Duration, warmupCount uint32) {
	// TODO: Override NumThreads for now, fix it later to support parallel threads
	//threads := test.ClientParam.NumThreads
	// TODO emit raw stats (e.g. sent/lost/received/lat per ping) and aggregate and emit results in a new go routine
	var wg sync.WaitGroup
	threads := uint32(1)
	for th := uint32(0); th < threads; th++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var sent, received, lost uint32
			latencyNumbers := make([]time.Duration, 0)
			for {
				select {
				case <-test.Done:
					result := payloads.NewLatencies(test, int(received), latencyNumbers)
					test.Results <- session.TestResult{
						Success: true,
						Error:   nil,
						Body: payloads.PingPayload{
							Latency:  result,
							Sent:     sent,
							Lost:     lost,
							Received: received,
						},
					}
					return
				default:
					t0 := time.Now()
					if warmupCount > 0 {
						warmupCount--
						_, _ = t.DoPing(test, "[warmup]")
					} else {
						sent++
						latency, err := t.DoPing(test, "")
						if err == nil {
							received++
							latencyNumbers = append(latencyNumbers, latency)
						} else {
							lost++
						}
					}
					// TODO add failure case. lost > received? all packets lost?
					if received >= 1000 {
						result := payloads.NewLatencies(test, int(received), latencyNumbers)
						test.Results <- session.TestResult{
							Success: true,
							Error:   nil,
							Body: payloads.PingPayload{
								Latency:  result,
								Sent:     sent,
								Lost:     lost,
								Received: received,
							},
						}
						latencyNumbers = make([]time.Duration, 0)
						sent, received, lost = 0, 0, 0
					}
					t1 := time.Since(t0)
					if t1 < g {
						time.Sleep(g - t1)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func (t Tests) DoPing(test *session.Test, prefix string) (time.Duration, error) {
	t0 := time.Now()
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort, 0, 0) // TODO force client port to 0?
	if err != nil {
		return 0, fmt.Errorf("%sconnection to %s timed out: %w", prefix, test.DialAddr, err)
	}
	timeTaken := time.Since(t0)
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		_ = tcpconn.SetLinger(0)
	}
	_ = conn.Close()
	return timeTaken, nil
}
