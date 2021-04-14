package session

import (
	"net"
	"sync"

	"weavelab.xyz/ethr/ethr"
)

type Conn struct {
	Conn net.Conn
	FD   uintptr
}

type Session struct {
	RemoteIP  string
	TestCount uint32
	Tests     map[TestID]*Test
}

var Logger ethr.Logger

var sessions = make(map[string]*Session)
var sessionLock sync.RWMutex

func GetSessions() []Session {
	out := make([]Session, 0, len(sessions))
	sessionLock.RLock()
	defer sessionLock.RUnlock()
	for _, v := range sessions {
		out = append(out, *v)
	}
	return out
}

func (s Session) CreateOrGetTest(rIP net.IP, rPort uint16, protocol ethr.Protocol, testType TestType, aggregator ResultAggregator) (*Test, bool) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	isNew := false
	test := getTest(rIP, protocol, testType)
	if test == nil {
		isNew = true
		test, _ = s.unsafeNewTest(rIP, rPort, protocol, testType, ethr.ClientParams{}, aggregator)
		test.IsActive = true
	}
	return test, isNew
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
	if len(s.Tests) == 0 {
		delete(sessions, s.RemoteIP)
	}
}

func (s Session) CreateTest(rIP net.IP, rPort uint16, protocol ethr.Protocol, tt TestType, clientParam ethr.ClientParams, aggregator ResultAggregator) (*Test, error) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	return s.unsafeNewTest(rIP, rPort, protocol, tt, clientParam, aggregator)
}

func (s Session) unsafeNewTest(rIP net.IP, rPort uint16, protocol ethr.Protocol, tt TestType, clientParam ethr.ClientParams, aggregator ResultAggregator) (*Test, error) {
	var session *Session
	session, found := sessions[rIP.String()]
	if !found {
		session = &Session{}
		session.RemoteIP = rIP.String()
		session.Tests = make(map[TestID]*Test)
		sessions[rIP.String()] = session
	}

	tID := TestID{
		Protocol: protocol,
		Type:     tt,
	}
	test, found := session.Tests[tID]
	if found {
		return test, nil
	}
	test = NewTest(&s, protocol, tt, rIP, rPort, clientParam, aggregator)
	session.Tests[tID] = test

	go test.StartPublishing()

	return test, nil
}

func getTest(remoteIP net.IP, proto ethr.Protocol, testType TestType) (test *Test) {
	test = nil
	session, found := sessions[remoteIP.String()]
	if !found {
		return
	}
	test, _ = session.Tests[TestID{Protocol: proto, Type: testType}]
	return
}
