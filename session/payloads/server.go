package payloads

type ServerPayload struct {
	PacketsPerSecond     uint64
	ConnectionsPerSecond uint64
	Bandwidth            uint64
	Latency              LatencyPayload
}
