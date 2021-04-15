package server

import (
	"context"
	"fmt"
	"time"
)

type ServerUI interface {
	Paint(uint64)
}

type UI struct {
	Terminal ServerUI

	TCP  *AggregateStats
	ICMP *AggregateStats
	UDP  *AggregateStats
}

func NewUI(terminalUI bool) *UI {
	var ui ServerUI
	var err error
	var tcp, udp, icmp AggregateStats
	if terminalUI {
		ui, err = InitTui(&tcp, &udp, &icmp)
		if err != nil {
			fmt.Println("Error: Failed to initialize UI.", err)
			fmt.Println("Using command line view instead of UI")
		}
	}

	if ui == nil {
		ui, _ = InitRawUI(&tcp, &udp, &icmp)
	}

	return &UI{
		Terminal: ui,
	}
}

func (u *UI) Display(ctx context.Context) {
	go func() {
		paintTicker := time.NewTicker(time.Second)
		start := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-paintTicker.C:
				seconds := uint64(time.Since(start).Seconds())
				u.Terminal.Paint(seconds)
				start = time.Now()
			}
		}
	}()
}
