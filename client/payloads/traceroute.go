package payloads

import (
	"net"
	"time"
)

type TraceRoutePayload struct {
	Hops []NetworkHop
}

type NetworkHop struct {
	Addr     net.Addr
	Sent     uint32
	Rcvd     uint32
	Lost     uint32
	Last     time.Duration
	Best     time.Duration
	Worst    time.Duration
	Average  time.Duration
	Total    time.Duration
	Name     string
	FullName string
}

func (h *NetworkHop) UpdateStats(peerAddr net.Addr, elapsed time.Duration) {
	h.Addr = peerAddr
	h.Rcvd++

	h.Last = elapsed
	h.Total += elapsed
	if h.Best > elapsed {
		h.Best = elapsed
	}
	if h.Worst < elapsed {
		h.Worst = elapsed
	}
	h.Average = time.Duration(h.Total.Nanoseconds() / int64(h.Rcvd))
}
