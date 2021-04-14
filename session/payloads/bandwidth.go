package payloads

type RawBandwidthPayload struct {
	ConnectionID     string
	Bandwidth        uint64
	PacketsPerSecond uint64
}

type BandwidthPayload struct {
	TotalBandwidth        uint64
	TotalPacketsPerSecond uint64
	ConnectionBandwidths  []RawBandwidthPayload
}
