package payloads

type ServerPayload struct {
	ConnectionsPerSecond uint64
	Bandwidth            uint64
	Latency              LatencyPayload
}
