package server

import (
	"context"
	"fmt"
	"time"
)

// TODO figure out a better way to interact with tui error/info panes
type ServerUI interface {
	Paint(uint64)
	AddInfoMsg(string)
	AddErrorMsg(string)
}

type UI struct {
	Terminal ServerUI
	isTui    bool

	TCP  *AggregateStats
	ICMP *AggregateStats
	UDP  *AggregateStats
}

func NewUI(terminalUI bool) *UI {
	var ui ServerUI
	var err error

	tcp, udp, icmp := NewAggregateStats(), NewAggregateStats(), NewAggregateStats()
	if terminalUI {
		ui, err = InitTui(tcp, udp, icmp)
		if err != nil {
			fmt.Println("Error: Failed to initialize UI.", err)
			fmt.Println("Using command line view instead of UI")
		}
	}

	if ui == nil {
		terminalUI = false
		ui, _ = InitRawUI(tcp, udp, icmp)
	}

	return &UI{
		Terminal: ui,
		isTui:    terminalUI,

		TCP:  tcp,
		UDP:  udp,
		ICMP: icmp,
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
