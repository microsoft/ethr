package tcp

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

type Handler struct {
	logger ethr.Logger
}

func NewHandler(logger ethr.Logger) Handler {
	return Handler{
		logger: logger,
	}
}

func (h Handler) HandleConn(ctx context.Context, test *session.Test, conn net.Conn) {
	defer conn.Close()

	test.AddIntermediateResult(session.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.ConnectionsPerSecondPayload{Connections: 1},
	})

	testID, clientParam, err := test.Session.HandshakeWithClient(conn)
	if err != nil {
		//// For ConnectionsPerSecond and Ping tests, there is no deterministic way to know when the test starts
		//// from the client side and when it ends. This defer function ensures that test is not
		//// created/deleted repeatedly by doing a deferred deletion. If another connection
		//// comes with-in 2s, then another reference would be taken on existing test object
		//// and it won't be deleted by safeDeleteTest call. This also ensures, test header is
		//// not printed repeatedly via emitTestHdr.
		//// Note: Similar mechanism is used in UDP tests to handle test lifetime as well.
		if operr, ok := err.(*net.OpError); ok && errors.Is(operr.Err, syscall.ECONNRESET) {
			// TODO find a better way to avoid spinning up go routines just to close them for all but the first connection
			go test.Session.PollInactive(ctx, 100*time.Millisecond)
			return
		}

		h.logger.Error("Failed in handshake with the client. Error: %v", err)
		return
	}
	if testID.Protocol == ethr.TCP {
		if testID.Type == ethr.TestTypeBandwidth {
			_ = h.TestBandwidth(ctx, test, clientParam, conn)
		} else if testID.Type == ethr.TestTypeLatency {
			_ = h.TestLatency(ctx, test, clientParam, conn)
		}
		session.DeleteTest(test)
	}
}

func ServerAggregator(nanos uint64, intermediateResults []session.TestResult) session.TestResult {
	connections := uint64(0)
	totalBandwidth := uint64(0)
	latencies := make([]time.Duration, 0, 1024) // TODO figure out reasonable initial capacity to avoid to many resizes

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
			ConnectionsPerSecond: 1e9 * connections / nanos,
			Bandwidth:            1e9 * totalBandwidth / nanos,
			Latency:              payloads.NewLatencies(latencies),
		},
	}
}
