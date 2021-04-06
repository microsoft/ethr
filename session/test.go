package session

import (
	"container/list"
	"net"
	"syscall"
	"time"
	"weavelab.xyz/ethr/ethr"
)

type TestType uint32

const (
	TestTypeAll TestType = iota
	TestTypeBandwidth
	TestTypeCps
	TestTypePps
	TestTypeLatency
	TestTypePing
	TestTypeTraceRoute
	TestTypeMyTraceRoute
)


type TestID struct {
	Protocol ethr.Protocol
	Type     TestType
}

type TestResult struct {
	Bandwidth            uint64
	ConnectionsPerSecond uint64
	PacketsPerSecond     uint64
	Latency              uint64
	// clatency uint64
}

type LatencyResult struct {
	RemoteIP string
	Protocol ethr.Protocol
	Avg time.Duration
	Min time.Duration
	Max time.Duration
	P50 time.Duration
	P90 time.Duration
	P95 time.Duration
	P99 time.Duration
	P999 time.Duration
	P9999 time.Duration
}

type BandwidthResult struct {

}

type Test struct {
	ID      TestID
	IsActive    bool
	IsDormant   bool
	Session     *Session
	RemoteAddr  string
	RemoteIP    string
	RemotePort  string
	DialAddr    string
	RefCount    int32
	ClientParam ethr.ClientParams
	Result  TestResult
	Done        chan struct{}
	ConnList    *list.List
	LastAccess  time.Time
}

func TestTypeToString(tt TestType) string {
	switch tt {
	case TestTypeBandwidth:
		return "Bandwidth"
	case TestTypeCps:
		return "Connections/s"
	case TestTypePps:
		return "Packets/s"
	case TestTypeLatency:
		return "Latency"
	case TestTypePing:
		return "Ping"
	case TestTypeTraceRoute:
		return "TraceRoute"
	case TestTypeMyTraceRoute:
		return "MyTraceRoute"
	default:
		return "Invalid"
	}
}


//func (t *Test)SafeDelete() bool {
//	sessionLock.Lock()
//	defer sessionLock.Unlock()
//	if atomic.AddInt32(&t.RefCount, -1) == 0 {
//		// TODO fix cleanup
//		//t.Session.DeleteTest(t)
//		//deleteTestInternal(t)
//		return true
//	}
//	return false
//}

//func (t *Test) addRef() {
//	sessionLock.Lock()
//	defer sessionLock.Unlock()
//	// TODO: Since we already take lock, atomic is not needed. Fix this later.
//	atomic.AddInt32(&t.RefCount, 1)
//}

func (t *Test) newConn(conn net.Conn) (c *Conn) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	c = &Conn{}
	c.Test = t
	c.Conn = conn
	c.FD = getFd(conn)
	c.Elem = t.ConnList.PushBack(c)
	return
}

func getFd(conn net.Conn) uintptr {
	var fd uintptr
	var rc syscall.RawConn
	var err error
	switch ct := conn.(type) {
	case *net.TCPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	case *net.UDPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	default:
		return 0
	}
	fn := func(s uintptr) {
		fd = s
	}
	rc.Control(fn)
	return fd
}

func (t *Test) delConn(conn net.Conn) {
	for e := t.ConnList.Front(); e != nil; e = e.Next() {
		ec := e.Value.(*Conn)
		if ec.Conn == conn {
			t.ConnList.Remove(e)
			break
		}
	}
}

func (t *Test) connListDo(f func(*Conn)) {
	sessionLock.RLock()
	defer sessionLock.RUnlock()
	for e := t.ConnList.Front(); e != nil; e = e.Next() {
		ec := e.Value.(*Conn)
		f(ec)
	}
}
