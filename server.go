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
	"sync/atomic"
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
	delay := timeToNextTick()
	ethrMsg = createAckMsg(gCert, delay)
	err = sendSessionMsg(enc, ethrMsg)
	if err != nil {
		ui.printErr("handleRequest: send session message: %v", err)
		cleanupFunc()
		return
	}

	size := test.testParam.BufferSize
	buff := make([]byte, size)
	for i := uint32(0); i < test.testParam.BufferSize; i++ {
		buff[i] = byte(i)
	}
	for {
		var err error
		if test.testParam.Reverse {
			_, err = conn.Write(buff)
		} else {
			_, err = io.ReadFull(conn, buff)
		}
		if err != nil {
			ui.printDbg("Error sending/receiving data on a connection for bandwidth test: %v", err)
			break
		}
		atomic.AddUint64(&test.testResult.bandwidth, uint64(size))
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
