//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"
)

func runXServer(testParam EthrTestParam, serverParam ethrServerParam) {
	defer stopStatsTimer()
	initXServer(serverParam.showUI)
	xsRunTCPServer()
	// runHTTPBandwidthServer()
	// runHTTPSBandwidthServer()
	startStatsTimer()
	toStop := make(chan int, 1)
	handleInterrupt(toStop)
	<-toStop
	ui.printMsg("Ethr done, received interrupt signal.")
}

func initXServer(showUI bool) {
	initServerUI(showUI)
}

func finiXServer() {
	ui.fini()
	logFini()
}

func xsRunTCPServer() {
	l, err := net.Listen(tcp(ipVer), hostAddr+":"+tcpBandwidthPort)
	if err != nil {
		finiXServer()
		fmt.Printf("Fatal error listening on "+tcpBandwidthPort+" for TCP tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpBandwidthPort + " for TCP tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printErr("xsRunTCPServer: error accepting new TCP connection: %v", err)
				continue
			}
			go xserverTCPHandler(conn)
		}
	}(l)
}

func xsCloseConn(conn net.Conn, cpsTest, bwTest *ethrTest) {
	err := conn.Close()
	if err != nil {
		ui.printDbg("Failed to close TCP connection, error: %v", err)
	}
	// Delay delete the test. This is to ensure that tests like CPS don't
	// end up not printing stats
	time.Sleep(2 * time.Second)
	if cpsTest != nil {
		safeDeleteTest(cpsTest)
	}
	if bwTest != nil {
		safeDeleteTest(bwTest)
	}
}

func xserverTCPHandler(conn net.Conn) {
	server, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	cpsTest, isNew := createOrGetTest(server, TCP, Cps)
	if cpsTest != nil {
		atomic.AddUint64(&cpsTest.testResult.data, 1)
	}
	if isNew {
		ui.emitTestHdr()
	}
	bwTest, _ := createOrGetTest(server, TCP, Bandwidth)
	defer xsCloseConn(conn, cpsTest, bwTest)
	buff := make([]byte, 2048)
	for {
		size, err := conn.Read(buff)
		if err != nil {
			return
		}
		if bwTest != nil {
			atomic.AddUint64(&bwTest.testResult.data, uint64(size))
		}
	}
}
