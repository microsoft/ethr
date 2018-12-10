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
    "fmt"
)

type EthrTestType uint32

const (
	Bandwidth EthrTestType = iota
	Cps
	Pps
	Latency
)

type EthrProtocol uint32

const (
	Tcp EthrProtocol = iota
	Udp
	Http
	Https
	Icmp
)

type EthrTestId struct {
	Protocol EthrProtocol
	Type     EthrTestType
}

type EthrMsgType uint32

const (
	EthrInv EthrMsgType = iota
	EthrSyn
	EthrAck
	EthrFin
	EthrBgn
	EthrEnd
)

type EthrMsgVer uint32

type EthrMsg struct {
	Version EthrMsgVer
	Type    EthrMsgType
	Syn     *EthrMsgSyn
	Ack     *EthrMsgAck
	Fin     *EthrMsgFin
	Bgn     *EthrMsgBgn
	End     *EthrMsgEnd
}

type EthrMsgSyn struct {
	TestParam EthrTestParam
}

type EthrMsgAck struct {
}

type EthrMsgFin struct {
	Message string
}

type EthrMsgBgn struct {
	UdpPort string
}

type EthrMsgEnd struct {
	Message string
}

type EthrTestParam struct {
	TestId     EthrTestId
	NumThreads uint32
	BufferSize uint32
	RttCount   uint32
}

type ethrTestResult struct {
	data uint64
}

type ethrTest struct {
	isActive   bool
	session    *ethrSession
	ctrlConn   net.Conn
	enc        *gob.Encoder
	dec        *gob.Decoder
    rcvdMsgs   chan *EthrMsg
	testParam  EthrTestParam
	testResult ethrTestResult
	done       chan struct{}
	connList   *list.List
}

type ethrConn struct {
	test    *ethrTest
	conn    net.Conn
	elem    *list.Element
	fd      uintptr
	data    uint64
	retrans uint64
}

type ethrSession struct {
	remoteAddr string
	testCount  uint32
	tests      map[EthrTestId]*ethrTest
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
	var session *ethrSession
	session, found := gSessions[remoteAddr]
	if !found {
		session = &ethrSession{}
		session.remoteAddr = remoteAddr
		session.tests = make(map[EthrTestId]*ethrTest)
		gSessions[remoteAddr] = session
		gSessionKeys = append(gSessionKeys, remoteAddr)
	}

	test, found := session.tests[testParam.TestId]
	if found {
		return nil, os.ErrExist
	}
	session.testCount++
	test = &ethrTest{}
	test.session = session
	test.ctrlConn = conn
	test.enc = enc
	test.dec = dec
    test.rcvdMsgs = make(chan *EthrMsg)
	test.testParam = testParam
	test.done = make(chan struct{})
	test.connList = list.New()
	session.tests[testParam.TestId] = test

	return test, nil
}

func deleteTest(test *ethrTest) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	session := test.session
	testId := test.testParam.TestId
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
	delete(session.tests, testId)
	session.testCount--

	if session.testCount == 0 {
		deleteKey(session.remoteAddr)
		delete(gSessions, session.remoteAddr)
	}
}

func getTest(remoteAddr string, proto EthrProtocol, testType EthrTestType) (test *ethrTest) {
	test = nil
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	session, found := gSessions[remoteAddr]
	if !found {
		return
	}
	test, found = session.tests[EthrTestId{proto, testType}]
	return
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
            fmt.Println(ethrMsg)
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

func createAckMsg() (ethrMsg *EthrMsg) {
	ethrMsg = &EthrMsg{Version: 0, Type: EthrAck}
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
	ethrMsg.Bgn.UdpPort = port
	return
}
