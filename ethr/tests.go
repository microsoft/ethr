package ethr

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
	switch s {
	case "s":
		return TestTypeServer
	case "b":
		return TestTypeBandwidth
	case "c":
		return TestTypeConnectionsPerSecond
	case "p":
		return TestTypePacketsPerSecond
	case "l":
		return TestTypeLatency
	case "pi":
		return TestTypePing
	case "tr":
		return TestTypeTraceRoute
	case "mtr":
		return TestTypeMyTraceRoute
	}
	return TestTypeUnknown
}
