package ethr

import "strings"

type TestType uint32

const (
	TestTypeServer TestType = iota
	TestTypeBandwidth
	TestTypeConnectionsPerSecond
	TestTypePacketsPerSecond
	TestTypeLatency
	TestTypePing
	TestTypeTraceRoute
	TestTypeMyTraceRoute
	TestTypeUnknown
)

func (p TestType) MarshalJSON() ([]byte, error) {
	switch p {
	case TestTypeServer:
		return []byte("Server"), nil
	case TestTypeBandwidth:
		return []byte("Bandwidth"), nil
	case TestTypeConnectionsPerSecond:
		return []byte("ConnectionsPerSecond"), nil
	case TestTypePacketsPerSecond:
		return []byte("PacketsPerSecond"), nil
	case TestTypeLatency:
		return []byte("Latency"), nil
	case TestTypePing:
		return []byte("Ping"), nil
	case TestTypeTraceRoute:
		return []byte("TraceRoute"), nil
	case TestTypeMyTraceRoute:
		return []byte("MyTraceRoute"), nil
	}
	return []byte("UNKNOWN"), nil
}

func (p TestType) String() string {
	switch p {
	case TestTypeServer:
		return "Server"
	case TestTypeBandwidth:
		return "Bandwidth"
	case TestTypeConnectionsPerSecond:
		return "ConnectionsPerSecond"
	case TestTypePacketsPerSecond:
		return "PacketsPerSecond"
	case TestTypeLatency:
		return "Latency"
	case TestTypePing:
		return "Ping"
	case TestTypeTraceRoute:
		return "TraceRoute"
	case TestTypeMyTraceRoute:
		return "MyTraceRoute"
	}
	return "UNKNOWN"
}

func ParseTestType(s string) TestType {
	switch strings.ToUpper(s) {
	case "S":
		return TestTypeServer
	case "B":
		return TestTypeBandwidth
	case "C":
		return TestTypeConnectionsPerSecond
	case "P":
		return TestTypePacketsPerSecond
	case "L":
		return TestTypeLatency
	case "PI":
		return TestTypePing
	case "TR":
		return TestTypeTraceRoute
	case "MTR":
		return TestTypeMyTraceRoute
	}
	return TestTypeUnknown
}

type TestID struct {
	Protocol Protocol
	Type     TestType
}
