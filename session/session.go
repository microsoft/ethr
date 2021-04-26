package session

import (
	"context"
	"net"
	"sync"
	"time"

	"weavelab.xyz/ethr/ethr"
)

type Session struct {
	sync.RWMutex
	Tests   map[ethr.TestID]*Test
	polling bool
	done    chan struct{}
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
func (s *Session) PollInactive(ctx context.Context, gap time.Duration) {
	s.RLock()
	if s.polling {
		s.RUnlock()
		return
	}
	s.polling = true
	s.RUnlock()

	Logger.Debug("starting new polling for session")
	ticker := time.NewTicker(gap)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			// TODO make sure frequent locking doesn't block test creation (especially for high throughput like UDP Bandwidth)
			toDelete := make([]*Test, 0)
			s.RLock()
			for k, v := range s.Tests {
				//Logger.Debug("Found Test from server: %v, time: %v", k, v.LastAccess)
				// At 200ms of no activity, mark the test in-active so stats stop
				// printing.
				if time.Since(v.LastAccess) > (200 * time.Millisecond) {
					v.IsDormant = true
				}
				// At 2s of no activity, delete the test by assuming that client
				// has stopped.
				if time.Since(v.LastAccess) > (2 * time.Second) {
					Logger.Debug("Deleting test from server: %v, lastAccess: %v", k, v.LastAccess)
					toDelete = append(toDelete, v)
				}
			}
			s.RUnlock()
			for _, t := range toDelete {
				DeleteTest(t) // delete needs a write lock so handle externally from
			}
		}
	}
}

func CreateOrGetTest(rIP net.IP, rPort uint16, protocol ethr.Protocol, testType ethr.TestType, params ethr.ClientParams, aggregator ResultAggregator, publishInterval time.Duration) (*Test, bool) {
	isNew := false
	session := getOrCreateSession(rIP)
	test := session.getTest(protocol, testType)
	if test == nil {
		isNew = true
		test, _ = session.newTest(rIP, rPort, protocol, testType, params, aggregator, publishInterval)
		test.IsActive = true
	}
	test.LastAccess = time.Now()
	test.IsDormant = false
	return test, isNew
}

func DeleteTest(t *Test) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	if s, ok := sessions[t.RemoteIP.String()]; ok {
		s.Lock()
		delete(s.Tests, t.ID)
		s.Unlock()
		if len(s.Tests) == 0 {
			close(s.done)
			delete(sessions, t.RemoteIP.String())
		}
	}
}

func getOrCreateSession(rIP net.IP) *Session {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	session, found := sessions[rIP.String()]
	if !found {
		session = &Session{
			Tests: make(map[ethr.TestID]*Test),
			done:  make(chan struct{}),
		}
		sessions[rIP.String()] = session
	}
	return session
}

func (s *Session) newTest(rIP net.IP, rPort uint16, protocol ethr.Protocol, tt ethr.TestType, clientParam ethr.ClientParams, aggregator ResultAggregator, publishInterval time.Duration) (*Test, error) {
	Logger.Debug("New test created from %s:%d", rIP, rPort)
	test := NewTest(s, protocol, tt, rIP, rPort, clientParam, aggregator, publishInterval)
	s.Lock()
	s.Tests[test.ID] = test
	s.Unlock()

	go test.StartPublishing()

	return test, nil
}

func (s *Session) getTest(proto ethr.Protocol, testType ethr.TestType) (test *Test) {
	s.RLock()
	test, _ = s.Tests[ethr.TestID{Protocol: proto, Type: testType}]
	s.RUnlock()
	return
}
