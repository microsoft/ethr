//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"sync/atomic"
	"time"
)

func runServer(testParam EthrTestParam, showUI bool) {
	initServer(showUI)
	l := runControlChannel()
	defer l.Close()
	runTCPBandwidthServer()
	runTCPCpsServer()
	runTCPLatencyServer()
	go runHTTPBandwidthServer()
	startStatsTimer()
	for {
		conn, err := l.Accept()
		if err != nil {
			ui.printErr("Error accepting new control connection: %v", err)
			continue
		}
		go handleRequest(conn)
	}
	stopStatsTimer()
}

func initServer(showUI bool) {
	initServerUI(showUI)
}

func finiServer() {
	ui.fini()
	logFini()
}

func runControlChannel() net.Listener {
	l, err := net.Listen(protoTCP, hostAddr+":"+ctrlPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening for control connections: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + ctrlPort + " for control plane")
	return l
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	ethrMsg := recvSessionMsg(dec)
	if ethrMsg.Type != EthrSyn {
		return
	}
	testParam := ethrMsg.Syn.TestParam
	server, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		ui.printDbg("RemoteAddr: Split host port failed: %v", err)
		return
	}
	ethrUnused(port)
	lserver, lport, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		ui.printDbg("LocalAddr: Split host port failed: %v", err)
		return
	}
	ethrUnused(lserver, lport)
	ui.printMsg("New control connection from " + server + ", port " + port)
	ui.printMsg("Starting " + protoToString(testParam.TestID.Protocol) + " " +
		testToString(testParam.TestID.Type) + " test from " + server)
	test, err := newTest(server, conn, testParam, enc, dec)
	if err != nil {
		msg := "Rejected duplicate " + protoToString(testParam.TestID.Protocol) + " " +
			testToString(testParam.TestID.Type) + " test from " + server
		ui.printMsg(msg)
		ethrMsg = createFinMsg(msg)
		sendSessionMsg(enc, ethrMsg)
		return
	}
	cleanupFunc := func() {
		test.ctrlConn.Close()
		close(test.done)
		deleteTest(test)
	}
	ui.emitTestHdr()
	if test.testParam.TestID.Protocol == UDP {
		if test.testParam.TestID.Type == Bandwidth {
			err = runUDPBandwidthServer(test)
		} else if test.testParam.TestID.Type == Pps {
			err = runUDPPpsServer(test)
		}
		if err != nil {
			ui.printDbg("Error encounterd in running UDP test (%s): %v",
				testToString(testParam.TestID.Type), err)
			cleanupFunc()
			return
		}
	}
	// TODO: Enable this in future, right now there is not much value coming
	// from this.
	/**
		ethrMsg = createAckMsg()
		err = sendSessionMsg(enc, ethrMsg)
		if err != nil {
			ui.printErr("send session message: %v",err)
			cleanupFunc()
			return
		}
		ethrMsg = recvSessionMsg(dec)
		if ethrMsg.Type != EthrAck {
			cleanupFunc()
			return
		}
	    **/
	test.isActive = true
	waitForChannelStop := make(chan bool, 1)
	serverWatchControlChannel(test, waitForChannelStop)
	<-waitForChannelStop
	ui.printMsg("Ending " + testToString(testParam.TestID.Type) + " test from " + server)
	test.isActive = false
	cleanupFunc()
	if len(gSessionKeys) > 0 {
		ui.emitTestHdr()
	}
}

func serverWatchControlChannel(test *ethrTest, waitForChannelStop chan bool) {
	watchControlChannel(test, waitForChannelStop)
}

func runTCPBandwidthServer() {
	l, err := net.Listen(protoTCP, hostAddr+":"+tcpBandwidthPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpBandwidthPort+" for TCP bandwidth tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpBandwidthPort + " for TCP bandwidth tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printErr("Error accepting new bandwidth connection: %v", err)
				continue
			}
			server, port, _ := net.SplitHostPort(conn.RemoteAddr().String())
			test := getTest(server, TCP, Bandwidth)
			if test == nil {
				ui.printDbg("Received unsolicited TCP connection on port %s from %s port %s", tcpBandwidthPort, server, port)
				conn.Close()
				continue
			}
			go runTCPBandwidthHandler(conn, test)
		}
	}(l)
}

func closeConn(conn net.Conn) {
	ui.printDbg("Closing TCP connection: %v", conn)
	err := conn.Close()
	if err != nil {
		ui.printDbg("Failed to close TCP connection, error: %v", err)
	}
}

func runTCPBandwidthHandler(conn net.Conn, test *ethrTest) {
	defer closeConn(conn)
	size := test.testParam.BufferSize
	bytes := make([]byte, size)
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			_, err := io.ReadFull(conn, bytes)
			if err != nil {
				ui.printDbg("Error receiving data on a connection for bandwidth test: %v", err)
				continue
			}
			atomic.AddUint64(&test.testResult.data, uint64(size))
		}
	}
}

func runTCPCpsServer() {
	l, err := net.Listen(protoTCP, hostAddr+":"+tcpCpsPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpCpsPort+" for TCP conn/s tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpCpsPort + " for TCP conn/s tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				// This can happen a lot during load, hence don't log by
				// default.
				ui.printDbg("Error accepting new conn/s connection: %v", err)
				continue
			}
			go runTCPCpsHandler(conn)
		}
	}(l)
}

func runTCPCpsHandler(conn net.Conn) {
	defer conn.Close()
	server, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	test := getTest(server, TCP, Cps)
	if test != nil {
		atomic.AddUint64(&test.testResult.data, 1)
	}
}

func runTCPLatencyServer() {
	l, err := net.Listen(protoTCP, hostAddr+":"+tcpLatencyPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpLatencyPort+" for TCP latency tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpLatencyPort + " for TCP latency tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printErr("Error accepting new latency connection: %v", err)
				continue
			}
			server, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			test := getTest(server, TCP, Latency)
			if test == nil {
				conn.Close()
				continue
			}
			ui.emitLatencyHdr()
			go runTCPLatencyHandler(conn, test)
		}
	}(l)
}

func runTCPLatencyHandler(conn net.Conn, test *ethrTest) {
	defer conn.Close()
	bytes := make([]byte, test.testParam.BufferSize)
	// TODO Override buffer size to 1 for now. Evaluate if we need to allow
	// client to specify the buffer size in future.
	bytes = make([]byte, 1)
	rttCount := test.testParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	for {
		_, err := io.ReadFull(conn, bytes)
		if err != nil {
			ui.printDbg("Error receiving data for latency test: %v", err)
			return
		}
		for i := uint32(0); i < rttCount; i++ {
			s1 := time.Now()
			_, err = conn.Write(bytes)
			if err != nil {
				ui.printDbg("Error sending data for latency test: %v", err)
				return
			}
			_, err = io.ReadFull(conn, bytes)
			if err != nil {
				ui.printDbg("Error receiving data for latency test: %v", err)
				return
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
		atomic.SwapUint64(&test.testResult.data, uint64(elapsed.Nanoseconds()))
		avg := elapsed
		min := latencyNumbers[0]
		max := latencyNumbers[rttCount-1]
		p50 := latencyNumbers[((rttCountFixed*50)/100)-1]
		p90 := latencyNumbers[((rttCountFixed*90)/100)-1]
		p95 := latencyNumbers[((rttCountFixed*95)/100)-1]
		p99 := latencyNumbers[((rttCountFixed*99)/100)-1]
		p999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.9)/100)-1)]
		p9999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.99)/100)-1)]
		ui.emitLatencyResults(
			test.session.remoteAddr,
			protoToString(test.testParam.TestID.Protocol),
			avg, min, max, p50, p90, p95, p99, p999, p9999)
	}
}

func runUDPBandwidthServer(test *ethrTest) error {
	udpAddr, err := net.ResolveUDPAddr(protoUDP, hostAddr+":"+udpBandwidthPort)
	if err != nil {
		ui.printDbg("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP(protoUDP, udpAddr)
	if err != nil {
		ui.printDbg("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}
	go func(l *net.UDPConn) {
		defer l.Close()
		//
		// We use NumCPU here instead of NumThreads passed from client. The
		// reason is that for UDP, there is no connection, so all packets come
		// on same CPU, so it isn't clear if there are any benefits to running
		// more threads than NumCPU(). TODO: Evaluate this in future.
		//
		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPBandwidthHandler(test, l)
		}
		<-test.done
	}(l)
	return nil
}

func runUDPBandwidthHandler(test *ethrTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	n, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		n, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			ui.printDbg("Error receiving data from UDP for bandwidth test: %v", err)
			continue
		}
		ethrUnused(n)
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(server, UDP, Bandwidth)
		if test != nil {
			atomic.AddUint64(&test.testResult.data, uint64(n))
		} else {
			ui.printDbg("Received unsolicited UDP traffic on port %s from %s port %s", udpPpsPort, server, port)
		}
	}
}

func runUDPPpsServer(test *ethrTest) error {
	udpAddr, err := net.ResolveUDPAddr(protoUDP, hostAddr+":"+udpPpsPort)
	if err != nil {
		ui.printDbg("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP(protoUDP, udpAddr)
	if err != nil {
		ui.printDbg("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}
	go func(l *net.UDPConn) {
		defer l.Close()
		//
		// We use NumCPU here instead of NumThreads passed from client. The
		// reason is that for UDP, there is no connection, so all packets come
		// on same CPU, so it isn't clear if there are any benefits to running
		// more threads than NumCPU(). TODO: Evaluate this in future.
		//
		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPPpsHandler(test, l)
		}
		<-test.done
	}(l)
	return nil
}

func runUDPPpsHandler(test *ethrTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	n, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		n, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			ui.printDbg("Error receiving data from UDP for pkt/s test: %v", err)
			continue
		}
		ethrUnused(n)
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(server, UDP, Pps)
		if test != nil {
			atomic.AddUint64(&test.testResult.data, 1)
		} else {
			ui.printDbg("Received unsolicited UDP traffic on port %s from %s port %s", udpPpsPort, server, port)
		}
	}
}

func runHTTPBandwidthServer() {
	http.HandleFunc("/", runHTTPBandwidthHandler)
	err := http.ListenAndServe(":"+httpBandwidthPort, nil)
	if err != nil {
		ui.printErr("Unable to start HTTP server, so HTTP tests cannot be run: %v", err)
	}
}

func runHTTPBandwidthHandler(w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ui.printDbg("Error reading HTTP body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		w.Write([]byte("ok"))
	case "PUT":
		w.Write([]byte("ok"))
	case "POST":
		w.Write([]byte("ok"))
	default:
		http.Error(w, "Only GET, PUT and POST are supported.", http.StatusMethodNotAllowed)
		return
	}
	server, _, _ := net.SplitHostPort(r.RemoteAddr)
	test := getTest(server, HTTP, Bandwidth)
	if test == nil {
		http.Error(w, "Unauthorized request.", http.StatusUnauthorized)
		return
	}
	if r.ContentLength > 0 {
		atomic.AddUint64(&test.testResult.data, uint64(r.ContentLength))
	}
}
