package payloads

import (
	"fmt"
	"net"
	"strings"
	"time"

	"weavelab.xyz/ethr/ui"
)

type TraceRoutePayload struct {
	Hops []NetworkHop
}

func (p TraceRoutePayload) String() string {
	parts := make([]string, 0, len(p.Hops))
	for _, hop := range p.Hops {
		parts = append(parts, hop.String())
	}
	return strings.Join(parts, "\n")
}

type NetworkHop struct {
	HopNumber int
	Addr      net.Addr
	Sent      uint32
	Rcvd      uint32
	Lost      uint32
	Last      time.Duration
	Best      time.Duration
	Worst     time.Duration
	Average   time.Duration
	Total     time.Duration
	Name      string
	FullName  string
}

func (h NetworkHop) String() string {
	if h.Addr == nil {
		return fmt.Sprintf("%2d | ???", h.HopNumber)
	}
	if h.FullName != "" {
		return fmt.Sprintf("%2d | %-70s\t%5d\t%5d\t%9s\t%9s\t%9s\t%9s", h.HopNumber, h.Addr.String()+"["+h.FullName+"]", h.Sent, h.Rcvd, ui.DurationToString(h.Last), ui.DurationToString(h.Average), ui.DurationToString(h.Best), ui.DurationToString(h.Worst))
	}
	return fmt.Sprintf("%2d | %-70s\t%5d\t%5d\t%9s\t%9s\t%9s\t%9s", h.HopNumber, h.Addr, h.Sent, h.Rcvd, ui.DurationToString(h.Last), ui.DurationToString(h.Average), ui.DurationToString(h.Best), ui.DurationToString(h.Worst))
}

func (h *NetworkHop) UpdateStats(peerAddr net.Addr, elapsed time.Duration) {
	h.Addr = peerAddr
	h.Rcvd++

	h.Last = elapsed
	h.Total += elapsed
	if h.Best > elapsed || h.Best == 0 {
		h.Best = elapsed
	}
	if h.Worst < elapsed {
		h.Worst = elapsed
	}
	h.Average = time.Duration(h.Total.Nanoseconds() / int64(h.Rcvd))
}
