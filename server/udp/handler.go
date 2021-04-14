package udp

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

func (h Handler) HandleConn(conn *net.UDPConn) {
	// For UDP, allocate buffer that can accomodate largest UDP datagram.
	readBuffer := make([]byte, 64*1024)

	var err error
	n := 0
	for err == nil {
		n, _, err = conn.ReadFrom(readBuffer) // don't actually care about the packet just how many bytes we read 'n'
		if err != nil {
			h.logger.Debug("Error receiving data from UDP for bandwidth test: %v", err)
			continue
		}

		if udpAddr, ok := conn.RemoteAddr().(*net.UDPAddr); ok {
			test, isNew := h.session.CreateOrGetTest(udpAddr.IP, uint16(udpAddr.Port), ethr.UDP, session.TestTypeServer, ServerAggregator)

			if isNew {
				h.logger.Debug("Creating UDP test from server: %v, lastAccess: %v", udpAddr.String(), time.Now())
			}

			if test != nil {
				test.IsDormant = false
				test.LastAccess = time.Now()
				test.AddIntermediateResult(session.TestResult{
					Success: true,
					Error:   nil,
					Body: payloads.RawBandwidthPayload{
						Bandwidth:        uint64(n),
						PacketsPerSecond: 1,
					},
				})
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
