package server

import "fmt"

type ServerUI interface {
	Paint(uint64)
}

type UI struct {
	Terminal ServerUI

	TCP  *AggregateStats
	ICMP *AggregateStats
	UDP  *AggregateStats
}

func NewUI(terminalUI bool) (*UI, error) {
	var ui ServerUI
	var err error
	if terminalUI {
		ui, err = InitTui()
		if err != nil {
			fmt.Println("Error: Failed to initialize UI.", err)
			fmt.Println("Using command line view instead of UI")
		}
	}

	if ui == nil {
		ui, _ = InitRawUI()
	}

	return &UI{
		Terminal: ui,
	}, nil
}
