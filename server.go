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
	"net"
	"os"
	"sort"
	"sync/atomic"
	"time"
)

var gCert []byte

func runServer(testParam EthrTestParam, serverParam ethrServerParam) {
	defer stopStatsTimer()
	initServer(serverParam.showUI)
	showAcceptedIPVersion()
	l := runControlChannel()
	defer l.Close()
	startStatsTimer()
	for {
		conn, err := l.Accept()
		if err != nil {
			ui.printErr("runServer: error accepting new control connection: %v", err)
			continue
		}
		go handleRequest(conn)
	}
}

func initServer(showUI bool) {
	initServerUI(showUI)
}

func finiServer() {
	ui.fini()
	logFini()
}

func runControlChannel() net.Listener {
	l, err := net.Listen(tcp(ipVer), hostAddr+":"+ctrlPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening for TCP connections: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + ctrlPort + " for control plane")
	return l
}

var gCps uint64 = 0

func handleRequest(conn net.Conn) {
	atomic.AddUint64(&gCps, 1)
	defer conn.Close()

	server, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	ethrUnused(port)
	if err != nil {
		ui.printDbg("RemoteAddr: Split host port failed: %v", err)
		return
	}

	test, _ := createOrGetTest(server, TCP, All)
	if test == nil {
		return
	}
	// ui.printMsg("Test: %v", test)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		safeDeleteTest(test)
	}()

	//
	// Always increment CPS count and then check if the test is Bandwidth
	// etc. and handle those cases as well.
	//
	atomic.AddUint64(&test.testResult.cps, 1)

	//
	// Check if there is any control message being sent to indicate type
	// of test, the client is running.
	//
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	ethrMsg := recvSessionMsg(dec)
	if ethrMsg.Type != EthrSyn {
		return
	}

	testParam := ethrMsg.Syn.TestParam

	lserver, lport, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		ui.printDbg("LocalAddr: Split host port failed: %v", err)
		return
	}
	ethrUnused(lserver, lport)

	ui.printMsg("New connection from " + server + ", port " + port)
	/*
		ui.printMsg("Starting " + protoToString(testParam.TestID.Protocol) + " " +
			testToString(testParam.TestID.Type) + " test from " + server)
	*/
	ui.emitTestHdr()
	delay := timeToNextTick()
	ethrMsg = createAckMsg(gCert, delay)
	err = sendSessionMsg(enc, ethrMsg)
	if err != nil {
		ui.printErr("handleRequest: Send session message failed: %v", err)
		return
	}
	test.isActive = true
	if testParam.TestID.Protocol == TCP {
		if testParam.TestID.Type == Bandwidth {
			runSrvrTCPBandwidthTest(test, testParam, conn)
		} else if testParam.TestID.Type == Latency {
			ui.emitLatencyHdr()
			runSrvrTCPLatencyTest(test, testParam, conn)
		}
	}
}

func runSrvrTCPBandwidthTest(test *ethrTest, testParam EthrTestParam, conn net.Conn) {
	size := testParam.BufferSize
	buff := make([]byte, size)
	for i := uint32(0); i < testParam.BufferSize; i++ {
		buff[i] = byte(i)
	}
	for {
		var err error
		if testParam.Reverse {
			_, err = conn.Write(buff)
		} else {
			_, err = io.ReadFull(conn, buff)
		}
		if err != nil {
			ui.printDbg("Error sending/receiving data on a connection for bandwidth test: %v", err)
			break
		}
		atomic.AddUint64(&test.testResult.bw, uint64(size))
	}
}

func runSrvrTCPLatencyTest(test *ethrTest, testParam EthrTestParam, conn net.Conn) {
	bytes := make([]byte, testParam.BufferSize)
	rttCount := testParam.RttCount
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
		atomic.SwapUint64(&test.testResult.latency, uint64(elapsed.Nanoseconds()))
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

func showAcceptedIPVersion() {
	var ipVerString = "ipv4, ipv6"
	if ipVer == ethrIPv4 {
		ipVerString = "ipv4"
	} else if ipVer == ethrIPv6 {
		ipVerString = "ipv6"
	}
	ui.printMsg("Accepting IP version: %s", ipVerString)
}
