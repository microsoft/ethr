//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"encoding/gob"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type EthrTestType uint32

const (
	All EthrTestType = iota
	Bandwidth
	Cps
	Pps
	Latency
	Ping
	TraceRoute
	MyTraceRoute
)

type EthrProtocol uint32

const (
	TCP EthrProtocol = iota
	UDP
	ICMP
)

const (
	ICMPv4 = 1  // ICMP for IPv4
	ICMPv6 = 58 // ICMP for IPv6
)

type EthrTestID struct {
	Protocol EthrProtocol
	Type     EthrTestType
}

type EthrMsgType uint32

const (
	EthrInv EthrMsgType = iota
	EthrSyn
	EthrAck
)

type EthrMsgVer uint32

type EthrMsg struct {
	Version EthrMsgVer
	Type    EthrMsgType
	Syn     *EthrMsgSyn
	Ack     *EthrMsgAck
}

type EthrMsgSyn struct {
	TestID      EthrTestID
	ClientParam EthrClientParam
}

type EthrMsgAck struct {
}

type ethrTestResult struct {
	bw      uint64
	cps     uint64
	pps     uint64
	latency uint64
	// clatency uint64
}

type ethrTest struct {
	isActive    bool
	isDormant   bool
	session     *ethrSession
	remoteAddr  string
	remoteIP    string
	remotePort  string
	dialAddr    string
	refCount    int32
	testID      EthrTestID
	clientParam EthrClientParam
	testResult  ethrTestResult
	done        chan struct{}
	connList    *list.List
	lastAccess  time.Time
}

type ethrIPVer uint32

const (
	ethrIPAny ethrIPVer = iota
	ethrIPv4
	ethrIPv6
)

type EthrClientParam struct {
	NumThreads  uint32
	BufferSize  uint32
	RttCount    uint32
	Reverse     bool
	Duration    time.Duration
	Gap         time.Duration
	WarmupCount uint32
	BwRate      uint64
	ToS         uint8
}

type ethrServerParam struct {
	showUI bool
}

var gIPVersion ethrIPVer = ethrIPAny
var gIsExternalClient bool

type ethrConn struct {
	bw      uint64
	pps     uint64
	test    *ethrTest
	conn    net.Conn
	elem    *list.Element
	fd      uintptr
	retrans uint64
}

type ethrSession struct {
	remoteIP  string
	testCount uint32
	tests     map[EthrTestID]*ethrTest
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

func newTest(remoteIP string, testID EthrTestID, clientParam EthrClientParam) (*ethrTest, error) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	return newTestInternal(remoteIP, testID, clientParam)
}

func newTestInternal(remoteIP string, testID EthrTestID, clientParam EthrClientParam) (*ethrTest, error) {
	var session *ethrSession
	session, found := gSessions[remoteIP]
	if !found {
		session = &ethrSession{}
		session.remoteIP = remoteIP
		session.tests = make(map[EthrTestID]*ethrTest)
		gSessions[remoteIP] = session
		gSessionKeys = append(gSessionKeys, remoteIP)
	}

	test, found := session.tests[testID]
	if found {
		return test, os.ErrExist
	}
	session.testCount++
	test = &ethrTest{}
	test.session = session
	test.refCount = 0
	test.testID = testID
	test.clientParam = clientParam
	test.done = make(chan struct{})
	test.connList = list.New()
	test.lastAccess = time.Now()
	test.isDormant = true
	session.tests[testID] = test

	return test, nil
}

func deleteTest(test *ethrTest) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	deleteTestInternal(test)
}

func deleteTestInternal(test *ethrTest) {
	session := test.session
	testID := test.testID
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
		deleteKey(session.remoteIP)
		delete(gSessions, session.remoteIP)
	}
}

func getTest(remoteIP string, proto EthrProtocol, testType EthrTestType) (test *ethrTest) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	return getTestInternal(remoteIP, proto, testType)
}

func getTestInternal(remoteIP string, proto EthrProtocol, testType EthrTestType) (test *ethrTest) {
	test = nil
	session, found := gSessions[remoteIP]
	if !found {
		return
	}
	test, _ = session.tests[EthrTestID{proto, testType}]
	return
}

func createOrGetTest(remoteIP string, proto EthrProtocol, testType EthrTestType) (test *ethrTest, isNew bool) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	isNew = false
	test = getTestInternal(remoteIP, proto, testType)
	if test == nil {
		isNew = true
		testID := EthrTestID{proto, testType}
		test, _ = newTestInternal(remoteIP, testID, EthrClientParam{})
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

func createSynMsg(testID EthrTestID, clientParam EthrClientParam) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrSyn}
	ethrMsg.Syn = &EthrMsgSyn{}
	ethrMsg.Syn.TestID = testID
	ethrMsg.Syn.ClientParam = clientParam
	return
}

func createAckMsg() (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrAck}
	ethrMsg.Ack = &EthrMsgAck{}
	return
}

func recvSessionMsg(conn net.Conn) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{}
	ethrMsg.Type = EthrInv
	msgBytes := make([]byte, 4)
	_, err := io.ReadFull(conn, msgBytes)
	if err != nil {
		ui.printDbg("Error receiving message on control channel. Error: %v", err)
		return
	}
	msgSize := binary.BigEndian.Uint32(msgBytes[0:])
	// TODO: Assuming max ethr message size as 16K sent over gob.
	if msgSize > 16384 {
		return
	}
	msgBytes = make([]byte, msgSize)
	_, err = io.ReadFull(conn, msgBytes)
	if err != nil {
		ui.printDbg("Error receiving message on control channel. Error: %v", err)
		return
	}
	ethrMsg = decodeMsg(msgBytes)
	return
}

func recvSessionMsgFromBuffer(msgBytes []byte) (ethrMsg *EthrMsg) {
	ethrMsg = decodeMsg(msgBytes)
	return
}

func sendSessionMsg(conn net.Conn, ethrMsg *EthrMsg) (err error) {
	msgBytes, err := encodeMsg(ethrMsg)
	if err != nil {
		ui.printDbg("Error sending message on control channel. Message: %v, Error: %v", ethrMsg, err)
		return
	}
	msgSize := len(msgBytes)
	tempBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBuf[0:], uint32(msgSize))
	_, err = conn.Write(tempBuf)
	if err != nil {
		ui.printDbg("Error sending message on control channel. Message: %v, Error: %v", ethrMsg, err)
	}
	_, err = conn.Write(msgBytes)
	if err != nil {
		ui.printDbg("Error sending message on control channel. Message: %v, Error: %v", ethrMsg, err)
	}
	return err
}

func decodeMsg(msgBytes []byte) (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{}
	buffer := bytes.NewBuffer(msgBytes)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(ethrMsg)
	if err != nil {
		ui.printDbg("Failed to decode message using Gob: %v", err)
		ethrMsg.Type = EthrInv
	}
	return
}

func encodeMsg(ethrMsg *EthrMsg) (msgBytes []byte, err error) {
	var writeBuffer bytes.Buffer
	encoder := gob.NewEncoder(&writeBuffer)
	err = encoder.Encode(ethrMsg)
	if err != nil {
		ui.printDbg("Failed to encode message using Gob: %v", err)
		return
	}
	msgBytes = writeBuffer.Bytes()
	return
}
