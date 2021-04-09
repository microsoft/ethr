package payloads

import (
	"net"
	"time"
)

type TraceRoutePayload struct {
	Hops []HopData
}

type HopData struct {
	Addr     net.Addr
	Sent     uint32
	Rcvd     uint32
	Lost     uint32
	Last     time.Duration
	Best     time.Duration
	Worst    time.Duration
	Total    time.Duration
	Name     string
	FullName string
}

func (h *HopData) UpdateStats(peerAddr net.Addr, elapsed time.Duration) {
	h.Addr = peerAddr
	h.Last = elapsed
	if h.Best > elapsed {
		h.Best = elapsed
	}
	if h.Worst < elapsed {
		h.Worst = elapsed
	}
	h.Total += elapsed
	h.Rcvd++
}
