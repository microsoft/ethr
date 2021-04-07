package payloads

type PingPayload struct {
	Latency  LatencyPayload
	Sent     uint32
	Lost     uint32
	Received uint32
}
