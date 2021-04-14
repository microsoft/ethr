package udp

import (
	"net"
	"sync/atomic"
	"time"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

type Handler struct {
	session session.Session
	logger  ethr.Logger
}

func (h Handler) HandleConn(conn *net.UDPConn) {
	// This local map aids in efficiency to look up a test based on client's IP
	// address. We could use createOrGetTest but that takes a global lock.
	// TODO move caching to session
	tests := make(map[string]*session.Test)
	// For UDP, allocate buffer that can accomodate largest UDP datagram.
	readBuffer := make([]byte, 64*1024)
	n, remoteIP, err := 0, new(net.UDPAddr), error(nil)

	// This function handles UDP tests that came from clients that are no longer
	// sending any traffic. This is poor man's garbage collection to ensure the
	// server doesn't end up printing dormant client related statistics as UDP
	// has no reliable way to detect if client is active or not.
	// TODO move to session handling
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			for k, v := range tests {
				h.logger.Debug("Found Test from server: %v, time: %v", k, v.LastAccess)
				// At 200ms of no activity, mark the test in-active so stats stop
				// printing.
				if time.Since(v.LastAccess) > (200 * time.Millisecond) {
					v.IsDormant = true
				}
				// At 2s of no activity, delete the test by assuming that client
				// has stopped.
				if time.Since(v.LastAccess) > (2 * time.Second) {
					h.logger.Debug("Deleting UDP test from server: %v, lastAccess: %v", k, v.LastAccess)
					h.session.DeleteTest(v.ID)
					delete(tests, k)
				}
			}
		}
	}()
	for err == nil {
		n, remoteIP, err = conn.ReadFromUDP(readBuffer)
		if err != nil {
			h.logger.Debug("Error receiving data from UDP for bandwidth test: %v", err)
			continue
		}
		//ethrUnused(remoteIP)
		//ethrUnused(n)
		//server, port, _ := net.SplitHostPort(remoteIP.String())
		server, _, _ := net.SplitHostPort(remoteIP.String())
		test, found := tests[server]
		if !found {
			test, isNew := h.session.CreateOrGetTest(server, ethr.UDP, session.TestTypeServer)
			if test != nil {
				tests[server] = test
			}
			if isNew {
				h.logger.Debug("Creating UDP test from server: %v, lastAccess: %v", server, time.Now())
				// TODO handle externally
				//ui.emitTestHdr()
			}
		}
		if test != nil {
			test.IsDormant = false
			test.LastAccess = time.Now()
			atomic.AddUint64(&test.Result.PacketsPerSecond, 1)
			atomic.AddUint64(&test.Result.Bandwidth, uint64(n))
		}
		//else {
		//h.logger.Debug("Unable to create test for UDP traffic on port %s from %s port %s", gEthrPortStr, server, port)
		//}
	}
}
