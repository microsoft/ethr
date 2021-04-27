package tcp

import (
	"net"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (t Tests) TestConnectionsPerSecond(test *session.Test) {
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		go func(th uint32) {
			for {
				select {
				case <-test.Done:
					return
				default:
					conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP, 0, 0, 0)
					if err != nil {
						//t.Logger.Debug("unable to dial TCP connection to %s: %w", test.DialAddr, err)
						continue
					}
					test.AddIntermediateResult(session.TestResult{
						Success: true,
						Error:   nil,
						Body:    payloads.ConnectionsPerSecondPayload{Connections: 1},
					})
					tcpconn, ok := conn.(*net.TCPConn)
					if ok {
						_ = tcpconn.SetLinger(0)
					}
					_ = conn.Close()
				}
			}
		}(th)
	}
}

func ConnectionsAggregator(microseconds uint64, intermediateResults []session.TestResult) session.TestResult {
	connections := uint64(0)
	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.ConnectionsPerSecondPayload); ok && r.Success {
			connections += body.Connections
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.ConnectionsPerSecondPayload{
			Connections: 1e6 * connections / microseconds,
		},
	}
}
