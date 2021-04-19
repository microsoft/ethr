package session

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"weavelab.xyz/ethr/ethr"
)

type Test struct {
	ID          ethr.TestID
	IsActive    bool
	IsDormant   bool
	Session     *Session
	RemoteIP    net.IP
	RemotePort  uint16
	DialAddr    string
	ClientParam ethr.ClientParams
	Results     chan TestResult
	Done        chan struct{}
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
		ID: ethr.TestID{
			Protocol: protocol,
			Type:     ttype,
		},
		RemoteIP:    rIP,
		RemotePort:  rPort,
		DialAddr:    dialAddr,
		ClientParam: params,
		Done:        make(chan struct{}),
		Results:     make(chan TestResult, 16), // TODO figure out appropriate buffer size (minimum 1 to avoid blocking an error)
		LastAccess:  time.Now(),
		IsDormant:   true,

		resultLock:          sync.Mutex{},
		intermediateResults: make([]TestResult, 0, 100),
		aggregator:          aggregator,
		latestResult:        TestResult{},
	}
}

func (t *Test) StartPublishing() {
	publishingGap := 100 * time.Millisecond // TODO how long to wait for?

	ticker := time.NewTicker(time.Second) // most metrics are per second
	// TODO figure out cleanup on test delete to avoid memory leak
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
			default:
				time.Sleep(publishingGap)
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
				t.intermediateResults = t.intermediateResults[:0]
				//t.intermediateResults = make([]TestResult, 0, cap(t.intermediateResults))
			}
			t.resultLock.Unlock()
			time.Sleep(publishingGap)
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
