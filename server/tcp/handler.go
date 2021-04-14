package tcp

import (
	"net"
	"time"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

type Handler struct {
	session session.Session
	logger  ethr.Logger
}

func (h Handler) HandleConn(test *session.Test, conn net.Conn) {
	defer conn.Close()

	isCPSorPing := true
	// For ConnectionsPerSecond and Ping tests, there is no deterministic way to know when the test starts
	// from the client side and when it ends. This defer function ensures that test is not
	// created/deleted repeatedly by doing a deferred deletion. If another connection
	// comes with-in 2s, then another reference would be taken on existing test object
	// and it won't be deleted by safeDeleteTest call. This also ensures, test header is
	// not printed repeatedly via emitTestHdr.
	// Note: Similar mechanism is used in UDP tests to handle test lifetime as well.
	defer func() {
		if isCPSorPing {
			time.Sleep(2 * time.Second) // must be longer than handshake timeout
		}
		h.session.DeleteTest(test.ID)
	}()

	// Always increment ConnectionsPerSecond count and then check if the test is Bandwidth etc. and handle
	// those cases as well.
	test.AddIntermediateResult(session.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.ConnectionsPerSecondPayload{Connections: 1},
	})

	testID, clientParam, err := h.session.HandshakeWithClient(conn)
	if err != nil {
		h.logger.Debug("Failed in handshake with the client. Error: %v", err)
		return
	}
	isCPSorPing = false
	if testID.Protocol == ethr.TCP {
		if testID.Type == session.TestTypeBandwidth {
			_ = h.TestBandwidth(test, clientParam, conn)
		} else if testID.Type == session.TestTypeLatency {
			_ = h.TestLatency(test, clientParam, conn)
		}
	}
}

func ServerAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	connections := uint64(0)
	totalBandwidth := uint64(0)
	latencies := make([]time.Duration, 0, 100) // TODO figure out reasonable initial capacity to avoid to many resizes

	for _, r := range intermediateResults {
		// ignore failed results

		switch body := r.Body.(type) {
		case payloads.BandwidthPayload:
			totalBandwidth += body.TotalBandwidth
		case payloads.RawLatencies:
			latencies = append(latencies, body.Latencies...)
		case payloads.ConnectionsPerSecondPayload:
			connections += body.Connections
		default:
			// do nothing, drop unknowns
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.ServerPayload{
			ConnectionsPerSecond: connections / seconds,
			Bandwidth:            totalBandwidth / seconds,
			Latency:              payloads.NewLatencies(len(latencies), latencies),
		},
	}
}
