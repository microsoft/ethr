package payloads

type BandwidthPayload struct {
	TotalBandwidth       uint64
	ConnectionBandwidths []uint64
	PacketsPerSecond     uint64
}
