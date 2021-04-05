package ethr

import "time"

type ClientParams struct {
	NumThreads  uint32
	BufferSize  uint32
	RttCount    uint32
	Reverse     bool
	Duration    time.Duration
	Gap         time.Duration
	WarmupCount uint32
	BwRate      uint64
	ToS         uint8
}

type ServerParams struct {
	showUI bool
}
