package udp

import (
	"context"
	"net"
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

func (h Handler) HandleConn(ctx context.Context, unused *session.Test, conn net.Conn) {
	// For UDP, allocate buffer that can accomodate largest UDP datagram.
	readBuffer := make([]byte, 64*1024)

	var err error
	for err == nil {
		if udpConn, ok := conn.(*net.UDPConn); ok {
			select {
			case <-ctx.Done():
				return
			default:
			}
			var raddr net.Addr
			var bytesRead int
			bytesRead, raddr, err = udpConn.ReadFrom(readBuffer) // don't actually care about the packet just how many bytesRead we read 'n'
			if err != nil {
				h.logger.Debug("Error receiving data from UDP for bandwidth test: %v", err)
				continue
			}

			if udpAddr, ok := raddr.(*net.UDPAddr); ok {
				test, isNew := session.CreateOrGetTest(udpAddr.IP, uint16(udpAddr.Port), ethr.UDP, ethr.TestTypeServer, ethr.ClientParams{}, ServerAggregator, time.Second)
				if isNew {
					h.logger.Debug("Creating UDP test from server: %v, lastAccess: %v", udpAddr.String(), time.Now())
					go test.Session.PollInactive(ctx, 100*time.Millisecond) // cleanup based on last access
				}

				if test != nil {
					test.AddIntermediateResult(session.TestResult{
						Success: true,
						Error:   nil,
						Body: payloads.RawBandwidthPayload{
							Bandwidth:        uint64(bytesRead),
							PacketsPerSecond: 1,
						},
					})
				}
			}
		}
	}
}

func ServerAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	totalBandwidth := uint64(0)
	totalPackets := uint64(0)

	for _, r := range intermediateResults {
		// ignore failed results

		switch body := r.Body.(type) {
		case payloads.RawBandwidthPayload:
			totalBandwidth += body.Bandwidth
			totalPackets += body.PacketsPerSecond
		default:
			// do nothing, drop unknowns
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.ServerPayload{
			PacketsPerSecond: totalPackets / seconds,
			Bandwidth:        totalBandwidth / seconds,
		},
	}
}
