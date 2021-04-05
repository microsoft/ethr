package tcp

import (
	"net"
	"os"
	"sync/atomic"
	"time"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)



type Handler struct {
	session ethr.Session
	logger ethr.Logger
}

func (h Handler) HandleConn(conn net.Conn) {
	defer conn.Close()

	server, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	//ethrUnused(server, port) // Seems like this is pointless
	if err != nil {
		h.logger.Debug("RemoteAddr: Split host port failed: %v", err)
		return
	}
	lserver, lport, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		h.logger.Debug("LocalAddr: Split host port failed: %v", err)
		return
	}
	//ethrUnused(lserver, lport)
	h.logger.Debug("New connection from %v, port %v to %v, port %v", server, port, lserver, lport)

	test, _ := h.session.CreateOrGetTest(server, ethr.TCP, ethr.TestTypeAll)
	if test == nil {
		return
	}

	// Should be handled in UI thread, signal?
	//if isNew {
	// TODO handle externally
	//	ui.emitTestHdr()
	//}

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
			time.Sleep(2 * time.Second)
		}
		h.session.SafeDeleteTest(test)
	}()

	// Always increment ConnectionsPerSecond count and then check if the test is Bandwidth etc. and handle
	// those cases as well.
	atomic.AddUint64(&test.Result.ConnectionsPerSecond, 1)

	testID, clientParam, err := h.handshakeWithClient(conn)
	if err != nil {
		h.logger.Debug("Failed in handshake with the client. Error: %v", err)
		return
	}
	isCPSorPing = false
	if testID.Protocol == ethr.TCP {
		if testID.Type == ethr.TestTypeBandwidth {
			h.TestBandwidth(test, clientParam, conn)
		} else if testID.Type == ethr.TestTypeLatency {
			// TODO Should be handled in UI thread, signal?
			//ui.emitLatencyHdr()
			h.TestLatency(test, clientParam, conn)
		}
	}
}

func (h Handler) handshakeWithClient(conn net.Conn) (testID ethr.TestID, clientParam ethr.ClientParams, err error) {
	msg := h.session.Receive(conn)
	if msg.Type != ethr.Syn {
		h.logger.Debug("Failed to receive SYN message from client.")
		err = os.ErrInvalid
		return
	}
	testID = msg.Syn.TestID
	clientParam = msg.Syn.ClientParam
	ack := session.CreateAckMsg()
	err = h.session.Send(conn, ack)
	return
}