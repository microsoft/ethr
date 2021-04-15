package session

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"syscall"
	"time"

	"weavelab.xyz/ethr/ethr"
)

type TestID struct {
	Protocol ethr.Protocol
	Type     ethr.TestType
}

type Test struct {
	ID          TestID
	IsActive    bool
	IsDormant   bool
	Session     *Session
	RemoteIP    net.IP
	RemotePort  uint16
	DialAddr    string
	ClientParam ethr.ClientParams
	Results     chan TestResult
	Done        chan struct{}
	ConnList    []*Conn
	LastAccess  time.Time

	resultLock          sync.Mutex
	intermediateResults []TestResult
	aggregator          ResultAggregator
	latestResult        TestResult
}

type TestResult struct {
	Success bool
	Error   error
	Body    interface{}
}

type ResultAggregator func(uint64, []TestResult) TestResult

func NewTest(s *Session, protocol ethr.Protocol, ttype ethr.TestType, rIP net.IP, rPort uint16, params ethr.ClientParams, aggregator ResultAggregator) *Test {
	dialAddr := fmt.Sprintf("[%s]:%s", rIP.String(), strconv.Itoa(int(rPort)))
	if protocol == ethr.ICMP {
		dialAddr = rIP.String()
	}
	return &Test{
		Session: s,
		ID: TestID{
			Protocol: protocol,
			Type:     ttype,
		},
		RemoteIP:    rIP,
		RemotePort:  rPort,
		DialAddr:    dialAddr,
		ClientParam: params,
		Done:        make(chan struct{}),
		Results:     make(chan TestResult, 16), // TODO figure out appropriate buffer size (minimum 1 to avoid blocking an error)
		ConnList:    make([]*Conn, 0, params.NumThreads),
		LastAccess:  time.Now(),
		IsDormant:   true,

		resultLock:          sync.Mutex{},
		intermediateResults: make([]TestResult, 0, 100),
		aggregator:          aggregator,
		latestResult:        TestResult{},
	}
}

func (t *Test) StartPublishing() {
	ticker := time.NewTicker(time.Second) // most metrics are per second
	for {
		start := time.Now()
		if t.aggregator != nil {
			select {
			case <-ticker.C:
				t.resultLock.Lock()
				if len(t.intermediateResults) == 0 {
					t.resultLock.Unlock()
					break
				}

				seconds := uint64(time.Since(start).Seconds())
				if seconds < 1 {
					seconds = 1
				}
				r := t.aggregator(seconds, t.intermediateResults)
				t.intermediateResults = make([]TestResult, 0, cap(t.intermediateResults))
				t.latestResult = r
				t.resultLock.Unlock()

				t.Results <- r

			}
		} else {
			t.resultLock.Lock()
			// TODO async publishing to avoid potential block? ordering wouldn't be guaranteed
			for _, r := range t.intermediateResults {
				t.Results <- r
				t.latestResult = r
			}
			if len(t.intermediateResults) > 0 {
				// TODO make sure old array is GC'ed
				t.intermediateResults = make([]TestResult, 0, cap(t.intermediateResults))
			}
			t.resultLock.Unlock()
			time.Sleep(100 * time.Millisecond) // TODO how long to wait for?
		}
	}

}

func (t *Test) AddIntermediateResult(r TestResult) {
	t.resultLock.Lock()
	defer t.resultLock.Unlock()
	t.intermediateResults = append(t.intermediateResults, r)
}

func (t *Test) LatestResult() TestResult {
	t.resultLock.Lock()
	defer t.resultLock.Unlock()
	return t.latestResult
}

func (t *Test) NewConn(conn net.Conn) (c *Conn) {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	c = &Conn{
		Conn: conn,
		FD:   getFd(conn),
	}
	t.ConnList = append(t.ConnList, c)
	return c
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

	// The docs say this pointer is not guaranteed to stay valid
	// https://pkg.go.dev/syscall#RawConn.Control
	// TODO find a better pattern for persistent access/interaction with fd
	fn := func(s uintptr) {
		fd = s
	}
	rc.Control(fn)
	return fd
}
