package session

import (
	"context"
	"net"
	"sync"
	"time"

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

// PollInactive handles UDP tests that came from clients that are no longer
// sending any traffic. This is poor man's garbage collection to ensure the
// server doesn't end up printing dormant client related statistics as UDP
// has no reliable way to detect if client is active or not.
func (s Session) PollInactive(ctx context.Context, gap time.Duration) {
	ticker := time.NewTicker(gap)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// TODO potential performance issue for VERY high volume of newly created tests
			sessionLock.Lock()
			for k, v := range s.Tests {
				Logger.Debug("Found Test from server: %v, time: %v", k, v.LastAccess)
				// At 200ms of no activity, mark the test in-active so stats stop
				// printing.
				if time.Since(v.LastAccess) > (200 * time.Millisecond) {
					v.IsDormant = true
				}
				// At 2s of no activity, delete the test by assuming that client
				// has stopped.
				if time.Since(v.LastAccess) > (2 * time.Second) {
					Logger.Debug("Deleting UDP test from server: %v, lastAccess: %v", k, v.LastAccess)
					s.unsafeDeleteTest(v.ID)
				}
			}
			sessionLock.Unlock()
		}
	}
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
	s.unsafeDeleteTest(id)
}

func (s Session) unsafeDeleteTest(id TestID) {
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
