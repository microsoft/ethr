package tcp

import (
	"net"
	"sync/atomic"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestConnectionsPerSecond(test *session.Test, results chan client.TestResult) {
	totalConnections := payloads.ConnectionsPerSecondPayload{}
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		go func(th uint32) {
			for {
				select {
				case <-test.Done:
					results <- client.TestResult{
						Success: false,
						Error:   nil,
						Body:    totalConnections,
					}
					return
				default:
					conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort, 0, 0) // TODO need to force local port to 0?
					if err != nil {
						//t.Logger.Debug("unable to dial TCP connection to %s: %w", test.DialAddr, err)
						continue
					}
					atomic.AddUint64(&totalConnections.Connections, 1)
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
