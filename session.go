//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"container/list"
	"encoding/gob"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// EthrTestType represents the test type.
type EthrTestType uint32

const (
	// All represents all tests - For now only applicable for servers
	All EthrTestType = iota

	// Bandwidth represents the bandwidth test.
	Bandwidth

	// Cps represents connections/s test.
	Cps

	// Pps represents packets/s test.
	Pps

	// Latency represents the latency test.
	Latency

	// ConnLatency represents connection setup latency.
	ConnLatency
)

// EthrProtocol represents the network protocol.
type EthrProtocol uint32

const (
	// TCP represents the tcp protocol.
	TCP EthrProtocol = iota

	// UDP represents the udp protocol.
	UDP

	// HTTP represents using http protocol.
	HTTP

	// HTTPS represents using https protocol.
	HTTPS

	// ICMP represents the icmp protocol.
	ICMP
)

// EthrTestID represents the test id.
type EthrTestID struct {
	// Protocol represents the protocol this test uses.
	Protocol EthrProtocol

	// Type represents the test type this test uses.
	Type EthrTestType
}

// EthrMsgType represents the message type.
type EthrMsgType uint32

const (
	// EthrInv represents the Inv message.
	EthrInv EthrMsgType = iota

	// EthrSyn represents the Syn message.
	EthrSyn

	// EthrAck represents the Ack message.
	EthrAck

	// EthrFin represents the Fin message.
	EthrFin

	// EthrBgn represents the Bgn message.
	EthrBgn

	// EthrEnd represents the End message.
	EthrEnd
)

// EthrMsgVer represents the message version.
type EthrMsgVer uint32

// EthrMsg represents the message entity.
type EthrMsg struct {
	// Version represents the message version.
	Version EthrMsgVer

	// Type represents the message type.
	Type EthrMsgType

	// Syn represents the Syn value.
	Syn *EthrMsgSyn

	// Ack represents the Ack value.
	Ack *EthrMsgAck

	// Fin represents the Fin value.
	Fin *EthrMsgFin

	// Bgn represents the Bgn value.
	Bgn *EthrMsgBgn

	// End represents the End value.
	End *EthrMsgEnd
}

// EthrMsgSyn represents the Syn entity.
type EthrMsgSyn struct {
	// TestParam represents the test parameters.
	TestParam EthrTestParam
}

// EthrMsgAck represents the Ack entity.
type EthrMsgAck struct {
	Cert []byte
}

// EthrMsgFin represents the Fin entity.
type EthrMsgFin struct {
	// Message represents the message body.
	Message string
}

// EthrMsgBgn represents the Bgn entity.
type EthrMsgBgn struct {
	// UDPPort represents the udp port.
	UDPPort string
}

// EthrMsgEnd represents the End entity.
type EthrMsgEnd struct {
	// Message represents the message body.
	Message string
}

// EthrTestParam represents the parameters used for the test.
type EthrTestParam struct {
	// TestID represents the test id of this test.
	TestID EthrTestID

	// NumThreads represents how many threads are used for the test.
	NumThreads uint32

	// BufferSize represents the buffer size.
	BufferSize uint32

	// RttCount represents the rtt count.
	RttCount uint32
}

type ethrTestResult struct {
	data uint64
}

type ethrTest struct {
	isActive   bool
	session    *ethrSession
	ctrlConn   net.Conn
	refCount   int32
	enc        *gob.Encoder
	dec        *gob.Decoder
	rcvdMsgs   chan *EthrMsg
	testParam  EthrTestParam
	testResult ethrTestResult
	done       chan struct{}
	connList   *list.List
}

type ethrMode uint32

const (
	ethrModeInv ethrMode = iota
	ethrModeServer
	ethrModeExtServer
	ethrModeClient
	ethrModeExtClient
)

type ethrIPVer uint32

const (
	ethrIPAny ethrIPVer = iota
	ethrIPv4
	ethrIPv6
)

type ethrClientParam struct {
	duration time.Duration
	gap      time.Duration
}

type ethrServerParam struct {
	showUI bool
}

var ipVer ethrIPVer = ethrIPAny

type ethrConn struct {
	data    uint64
	test    *ethrTest
	conn    net.Conn
	elem    *list.Element
	fd      uintptr
	retrans uint64
}

type ethrSession struct {
	remoteAddr string
	testCount  uint32
	tests      map[EthrTestID]*ethrTest
}

var gSessions = make(map[string]*ethrSession)
var gSessionKeys = make([]string, 0)
var gSessionLock sync.RWMutex

func deleteKey(key string) {
	i := 0
	for _, x := range gSessionKeys {
		if x != key {
			gSessionKeys[i] = x
			i++
		}
	}
	gSessionKeys = gSessionKeys[:i]
}

func newTest(remoteAddr string, conn net.Conn, testParam EthrTestParam, enc *gob.Encoder, dec *gob.Decoder) (*ethrTest, error) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	return newTestInternal(remoteAddr, conn, testParam, enc, dec)
}

func newTestInternal(remoteAddr string, conn net.Conn, testParam EthrTestParam, enc *gob.Encoder, dec *gob.Decoder) (*ethrTest, error) {
	var session *ethrSession
	session, found := gSessions[remoteAddr]
	if !found {
		session = &ethrSession{}
		session.remoteAddr = remoteAddr
		session.tests = make(map[EthrTestID]*ethrTest)
		gSessions[remoteAddr] = session
		gSessionKeys = append(gSessionKeys, remoteAddr)
	}

	test, found := session.tests[testParam.TestID]
	if found {
		return test, os.ErrExist
	}
	session.testCount++
	test = &ethrTest{}
	test.session = session
	test.ctrlConn = conn
	test.refCount = 0
	test.enc = enc
	test.dec = dec
	test.rcvdMsgs = make(chan *EthrMsg)
	test.testParam = testParam
	test.done = make(chan struct{})
	test.connList = list.New()
	session.tests[testParam.TestID] = test

	return test, nil
}

func deleteTest(test *ethrTest) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	deleteTestInternal(test)
}

func deleteTestInternal(test *ethrTest) {
	session := test.session
	testID := test.testParam.TestID
	//
	// TODO fix this, we need to decide where to close this, inside this
	// function or by the caller. The reason we may need it to be done by
	// the caller is, because done is used for test done notification and
	// there may be some time after done that consumers are still accessing it
	//
	// Since we have not added any refCounting on test object, we are doing
	// hacky timeout based solution by closing "done" outside and sleeping
	// for sufficient time. ugh!
	//
	// close(test.done)
	// test.ctrlConn.Close()
	// test.session = nil
	// test.connList = test.connList.Init()
	//
	delete(session.tests, testID)
	session.testCount--

	if session.testCount == 0 {
		deleteKey(session.remoteAddr)
		delete(gSessions, session.remoteAddr)
	}
}

func getTest(remoteAddr string, proto EthrProtocol, testType EthrTestType) (test *ethrTest) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	return getTestInternal(remoteAddr, proto, testType)
}

func getTestInternal(remoteAddr string, proto EthrProtocol, testType EthrTestType) (test *ethrTest) {
	test = nil
	session, found := gSessions[remoteAddr]
	if !found {
		return
	}
	test, _ = session.tests[EthrTestID{proto, testType}]
	return
}

func createOrGetTest(remoteAddr string, proto EthrProtocol, testType EthrTestType) (test *ethrTest, isNew bool) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	isNew = false
	test = getTestInternal(remoteAddr, proto, testType)
	if test == nil {
		isNew = true
		testParam := EthrTestParam{TestID: EthrTestID{proto, testType}}
		test, _ = newTestInternal(remoteAddr, nil, testParam, nil, nil)
		test.isActive = true
	}
	atomic.AddInt32(&test.refCount, 1)
	return
}

func safeDeleteTest(test *ethrTest) bool {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	if atomic.AddInt32(&test.refCount, -1) == 0 {
		deleteTestInternal(test)
		return true
	}
	return false
}

func addRef(test *ethrTest) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	// TODO: Since we already take lock, atomic is not needed. Fix this later.
	atomic.AddInt32(&test.refCount, 1)
}

func (test *ethrTest) newConn(conn net.Conn) (ec *ethrConn) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	ec = &ethrConn{}
	ec.test = test
	ec.conn = conn
	ec.fd = getFd(conn)
	ec.elem = test.connList.PushBack(ec)
	return
}

func (test *ethrTest) delConn(conn net.Conn) {
	for e := test.connList.Front(); e != nil; e = e.Next() {
		ec := e.Value.(*ethrConn)
		if ec.conn == conn {
			test.connList.Remove(e)
			break
		}
	}
}

func (test *ethrTest) connListDo(f func(*ethrConn)) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for e := test.connList.Front(); e != nil; e = e.Next() {
		ec := e.Value.(*ethrConn)
		f(ec)
	}
}

func watchControlChannel(test *ethrTest, waitForChannelStop chan bool) {
	go func() {
		for {
			ethrMsg := recvSessionMsg(test.dec)
			if ethrMsg.Type == EthrInv {
				break
			}
			test.rcvdMsgs <- ethrMsg
			ui.printDbg("%v", ethrMsg)
		}
		waitForChannelStop <- true
	}()
}

func recvSessionMsg(dec *gob.Decoder) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{}
	err := dec.Decode(ethrMsg)
	if err != nil {
		ui.printDbg("Error receiving message on control channel: %v", err)
		ethrMsg.Type = EthrInv
	}
	return
}

func sendSessionMsg(enc *gob.Encoder, ethrMsg *EthrMsg) error {
	err := enc.Encode(ethrMsg)
	if err != nil {
		ui.printDbg("Error sending message on control channel. Message: %v, Error: %v", ethrMsg, err)
	}
	return err
}

func createAckMsg(cert []byte) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrAck}
	ethrMsg.Ack = &EthrMsgAck{}
	ethrMsg.Ack.Cert = cert
	return
}

func createFinMsg(message string) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrFin}
	ethrMsg.Fin = &EthrMsgFin{}
	ethrMsg.Fin.Message = message
	return
}

func createSynMsg(testParam EthrTestParam) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrSyn}
	ethrMsg.Syn = &EthrMsgSyn{}
	ethrMsg.Syn.TestParam = testParam
	return
}

func createBgnMsg(port string) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrBgn}
	ethrMsg.Bgn = &EthrMsgBgn{}
	ethrMsg.Bgn.UDPPort = port
	return
}
