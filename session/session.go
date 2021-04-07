package session

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

	"weavelab.xyz/ethr/ethr"
)

type Conn struct {
	Bandwidth        uint64
	PacketsPerSecond uint64
	Test             *Test
	Conn             net.Conn
	Elem             *list.Element
	FD               uintptr
	Retransmits      uint64
}

type Session struct {
	RemoteIP  string
	TestCount uint32
	Tests     map[TestID]*Test
}

var Logger ethr.Logger

var sessions = make(map[string]*Session)
var sessionLock sync.RWMutex

func (s Session) CreateOrGetTest(remoteIP string, proto ethr.Protocol, testType TestType) (test *Test, isNew bool) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	isNew = false
	test = getTestInternal(remoteIP, proto, testType)
	if test == nil {
		isNew = true
		testID := TestID{Protocol: proto, Type: testType}
		test, _ = s.unsafeNewTest(remoteIP, testID, ethr.ClientParams{})
		test.IsActive = true
	}
	atomic.AddInt32(&test.RefCount, 1)
	return
}

func (s Session) DeleteTest(id TestID) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
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
	delete(s.Tests, id)
	s.TestCount--

	if s.TestCount == 0 {
		delete(sessions, s.RemoteIP)
	}
}

func (s Session) NewTest(remoteIP string, testID TestID, clientParam ethr.ClientParams) (*Test, error) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	return s.unsafeNewTest(remoteIP, testID, clientParam)
}

func (s Session) unsafeNewTest(remoteIP string, testID TestID, clientParam ethr.ClientParams) (*Test, error) {
	var session *Session
	session, found := sessions[remoteIP]
	if !found {
		session = &Session{}
		session.RemoteIP = remoteIP
		session.Tests = make(map[TestID]*Test)
		sessions[remoteIP] = session
	}

	test, found := session.Tests[testID]
	if found {
		return test, os.ErrExist
	}
	session.TestCount++
	test = &Test{}
	test.Session = session
	test.RefCount = 0
	test.ID = testID
	test.ClientParam = clientParam
	test.Done = make(chan struct{})
	test.ConnList = list.New()
	test.LastAccess = time.Now()
	test.IsDormant = true
	session.Tests[testID] = test

	return test, nil
}

func getTestInternal(remoteIP string, proto ethr.Protocol, testType TestType) (test *Test) {
	test = nil
	session, found := sessions[remoteIP]
	if !found {
		return
	}
	test, _ = session.Tests[TestID{Protocol: proto, Type: testType}]
	return
}

func (s Session) Receive(conn net.Conn) (msg *ethr.Msg) {
	msg = &ethr.Msg{}
	msg.Type = ethr.Inv
	msgBytes := make([]byte, 4)
	_, err := io.ReadFull(conn, msgBytes)
	if err != nil {
		Logger.Debug("Error receiving message on control channel. Error: %v", err)
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
		Logger.Debug("Error receiving message on control channel. Error: %v", err)
		return
	}
	msg = decodeMsg(msgBytes)
	return
}

func (s Session) ReceiveFromBuffer(msgBytes []byte) (msg *ethr.Msg) {
	msg = decodeMsg(msgBytes)
	return
}

func (s Session) Send(conn net.Conn, msg *ethr.Msg) (err error) {
	msgBytes, err := encodeMsg(msg)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
		return
	}
	msgSize := len(msgBytes)
	tempBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBuf[0:], uint32(msgSize))
	_, err = conn.Write(tempBuf)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
	}
	_, err = conn.Write(msgBytes)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
	}
	return err
}

func decodeMsg(msgBytes []byte) (msg *ethr.Msg) {
	msg = &ethr.Msg{}
	buffer := bytes.NewBuffer(msgBytes)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(msg)
	if err != nil {
		Logger.Debug("Failed to decode message using Gob: %v", err)
		msg.Type = ethr.Inv
	}
	return
}

func encodeMsg(msg *ethr.Msg) (msgBytes []byte, err error) {
	var writeBuffer bytes.Buffer
	encoder := gob.NewEncoder(&writeBuffer)
	err = encoder.Encode(msg)
	if err != nil {
		Logger.Debug("Failed to encode message using Gob: %v", err)
		return
	}
	msgBytes = writeBuffer.Bytes()
	return
}
