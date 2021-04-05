package ethr

import (
	"container/list"
	"time"
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
	Protocol Protocol
	Type     TestType
}

type TestResult struct {
	Bandwidth uint64
	CPS       uint64
	PPS       uint64
	Latency   uint64
	// clatency uint64
}

type LatencyResult struct {
	RemoteIP string
	Protocol Protocol
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
	ClientParam ClientParams
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