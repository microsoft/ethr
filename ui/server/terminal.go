package server

import (
	"fmt"
	"os"
	"sync"

	"weavelab.xyz/ethr/session/payloads"

	tm "github.com/nsf/termbox-go"
	"weavelab.xyz/ethr/config"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/stats"
	"weavelab.xyz/ethr/ui"
)

type Tui struct {
	tcpStats  *AggregateStats
	udpStats  *AggregateStats
	icmpStats *AggregateStats

	h, w                               int
	resX, resY, resW                   int
	latX, latY, latW                   int
	topVSplitX, topVSplitY, topVSplitH int
	statX, statY, statW                int
	msgX, msgY, msgW                   int
	botVSplitX, botVSplitY, botVSplitH int
	errX, errY, errW                   int
	res                                table
	results                            [][]string
	resultHdr                          []string
	msg                                table
	msgRing                            []string
	err                                table
	errRing                            []string
	ringLock                           sync.RWMutex
}

func InitTui(tcp *AggregateStats, udp *AggregateStats, icmp *AggregateStats) (*Tui, error) {
	err := tm.Init()
	if err != nil {
		return nil, err
	}

	w, h := tm.Size()
	//if h < 40 || w < 80 {
	//	tm.Close()
	//	s := fmt.Sprintf("Terminal too small (%dwx%dh), must be at least 40hx80w", w, h)
	//	return nil, errors.New(s)
	//}

	tm.SetInputMode(tm.InputEsc | tm.InputMouse)
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	tm.Sync()
	tm.Flush()
	tm.SetCursor(0, 0) // hide cursor

	botScnH := 8
	statScnW := 26
	resW := w - statScnW
	msgW := (w+1)/2 + 1

	tui := Tui{
		h:          h,
		w:          w,
		resX:       0,
		resY:       0,
		resW:       resW,
		latX:       0,
		latY:       h - botScnH,
		latW:       w,
		topVSplitY: 1,
		topVSplitH: h - botScnH,
		topVSplitX: resW,
		statY:      2,
		statW:      statScnW,
		statX:      resW + 1,
		msgX:       0,
		msgY:       h - botScnH + 1,
		msgW:       msgW,
		botVSplitX: msgW,
		botVSplitY: h - botScnH,
		botVSplitH: botScnH,
		errX:       msgW + 1,
		errY:       h - botScnH + 1,

		tcpStats:  tcp,
		udpStats:  udp,
		icmpStats: icmp,
	}

	tui.resultHdr = []string{"RemoteAddress", "Proto", "Bits/s", "Conn/s", "Pkts/s", "Avg Latency"}
	tui.results = make([][]string, 0)
	tui.res = table{
		6,
		[]int{13, 5, 7, 7, 7, 8},
		0,
		2,
		0,
		tableJustifyRight,
		tableNoBorder,
	}

	tui.msgRing = make([]string, botScnH-1)
	tui.msg = table{
		1,
		[]int{tui.msgW},
		tui.msgX,
		tui.msgY,
		0,
		tableJustifyLeft,
		tableNoBorder,
	}

	tui.errRing = make([]string, botScnH-1)
	tui.errW = w - tui.msgW - 1
	tui.err = table{
		1,
		[]int{tui.errW},
		tui.errX,
		tui.errY,
		0,
		tableJustifyLeft,
		tableNoBorder,
	}

	go func() {
		for {
			switch ev := tm.PollEvent(); ev.Type {
			case tm.EventKey:
				if ev.Key == tm.KeyEsc || ev.Key == tm.KeyCtrlC {
					tm.Close()
					os.Exit(0)
				}
			case tm.EventResize:
			}
		}
	}()

	return &tui, nil
}

// TODO change to milliseconds or smaller
func (t *Tui) Paint(nanos uint64) {
	if nanos < 1 {
		nanos = 1 // avoid divide by zero
	}
	_ = tm.Clear(tm.ColorDefault, tm.ColorDefault)
	defer tm.Flush()
	printCenterText(0, 0, t.w, "Ethr (Version: "+config.Version+")", tm.ColorBlack, tm.ColorWhite)
	printHLineText(t.resX, t.resY-1, t.resW, "Test Results")
	printHLineText(t.statX, t.statY-1, t.statW, "Statistics")
	printVLine(t.topVSplitX, t.topVSplitY, t.topVSplitH)

	printHLineText(t.msgX, t.msgY-1, t.msgW, "Messages")
	printHLineText(t.errX, t.errY-1, t.errW, "Errors")

	t.ringLock.Lock()
	t.msg.cr = 0
	for _, s := range t.msgRing {
		t.msg.addTblRow([]string{s})
	}

	t.err.cr = 0
	for _, s := range t.errRing {
		t.err.addTblRow([]string{s})
	}
	t.ringLock.Unlock()

	printVLine(t.botVSplitX, t.botVSplitY, t.botVSplitH)

	t.res.cr = 0
	sessions := session.GetSessions()
	if len(sessions) > 0 {
		t.res.addTblHdr()
		t.res.addTblRow(t.resultHdr)
		t.res.addTblSpr()
	}
	tcpActive, udpActive, icmpActive := false, false, false
	for _, s := range sessions {
		tcpResults := t.getTestResults(&s, ethr.TCP, t.tcpStats)
		if len(tcpResults) > 0 {
			t.res.addTblRow(tcpResults)
			t.res.addTblSpr()
			tcpActive = true
		}

		udpResults := t.getTestResults(&s, ethr.UDP, t.udpStats)
		if len(udpResults) > 0 {
			t.res.addTblRow(udpResults)
			t.res.addTblSpr()
			udpActive = true
		}

		icmpResults := t.getTestResults(&s, ethr.ICMP, t.icmpStats)
		if len(icmpResults) > 0 {
			t.res.addTblRow(icmpResults)
			t.res.addTblSpr()
			icmpActive = true
		}
	}

	if len(sessions) > 0 {
		if tcpActive {
			tcpAgg := t.tcpStats.ToString(ethr.TCP)
			t.tcpStats.Reset()
			t.res.addTblRow(tcpAgg)
			t.res.addTblSpr()
		}

		if udpActive {
			udpAgg := t.udpStats.ToString(ethr.UDP)
			t.udpStats.Reset()
			t.res.addTblRow(udpAgg)
			t.res.addTblSpr()
		}

		if icmpActive {
			icmpAgg := t.icmpStats.ToString(ethr.ICMP)
			t.icmpStats.Reset()
			t.res.addTblRow(icmpAgg)
			t.res.addTblSpr()
		}
	}

	previousStats := stats.PreviousStats()

	if len(previousStats.Devices) == 0 {
		return
	}

	currentStats := stats.LatestStats()

	x := t.statX
	w := t.statW
	y := t.statY
	for _, device := range currentStats.Devices {
		nsDiff := stats.DiffNetDevStats(device, previousStats, nanos)
		// TODO: Log the network adapter stats in file as well.
		printText(x, y, w, fmt.Sprintf("if: %s", device.InterfaceName), tm.ColorWhite, tm.ColorBlack)
		y++
		printText(x, y, w, fmt.Sprintf("Tx %sbps", ui.BytesToRate(nsDiff.TXBytes)), tm.ColorWhite, tm.ColorBlack)
		bw := nsDiff.TXBytes * 8
		PrintUsageBar(x+14, y, 10, bw, ui.KILO, tm.ColorYellow)
		y++
		printText(x, y, w, fmt.Sprintf("Rx %sbps", ui.BytesToRate(nsDiff.RXBytes)), tm.ColorWhite, tm.ColorBlack)
		bw = nsDiff.RXBytes * 8
		PrintUsageBar(x+14, y, 10, bw, ui.KILO, tm.ColorGreen)
		y++
		printText(x, y, w, fmt.Sprintf("Tx %spps", ui.NumberToUnit(nsDiff.TXPackets)), tm.ColorWhite, tm.ColorBlack)
		PrintUsageBar(x+14, y, 10, nsDiff.TXPackets, 10, tm.ColorWhite)
		y++
		printText(x, y, w, fmt.Sprintf("Rx %spps", ui.NumberToUnit(nsDiff.RXPackets)), tm.ColorWhite, tm.ColorBlack)
		PrintUsageBar(x+14, y, 10, nsDiff.RXPackets, 10, tm.ColorCyan)
		y++
		printText(x, y, w, "-------------------------", tm.ColorDefault, tm.ColorDefault)
		y++
	}
	printText(x, y, w,
		fmt.Sprintf("Tcp Retrans: %s",
			ui.NumberToUnit(1e9*(currentStats.TCP.RetransmittedSegments-previousStats.TCP.RetransmittedSegments)/nanos)),
		tm.ColorDefault, tm.ColorDefault)
}

func (t *Tui) getTestResults(s *session.Session, protocol ethr.Protocol, agg *AggregateStats) []string {
	var bwTestOn, cpsTestOn, ppsTestOn, latTestOn bool
	var bw, cps, pps uint64
	var lat payloads.LatencyPayload
	test, found := s.Tests[ethr.TestID{Protocol: protocol, Type: ethr.TestTypeServer}]
	if found && test.IsActive {
		result := test.LatestResult()
		if body, ok := result.Body.(payloads.ServerPayload); ok {
			bwTestOn = true
			bw = body.Bandwidth
			agg.Bandwidth += body.Bandwidth

			if protocol == ethr.TCP {
				cpsTestOn = true
				cps = body.ConnectionsPerSecond
				agg.ConnectionsPerSecond += body.ConnectionsPerSecond

				if len(body.Latency.Raw) > 0 {
					latTestOn = true
					lat = body.Latency
				}
			}

			if protocol == ethr.UDP {
				ppsTestOn = true
				pps = body.PacketsPerSecond
				agg.PacketsPerSecond += body.PacketsPerSecond
			}

		}

		if test.IsDormant && !bwTestOn && !cpsTestOn && !ppsTestOn && !latTestOn {
			return []string{}
		}
	}

	if bwTestOn || cpsTestOn || ppsTestOn || latTestOn {
		var bwStr, cpsStr, ppsStr, latStr string = "--  ", "--  ", "--  ", "--  "
		if bwTestOn {
			bwStr = ui.BytesToRate(bw)
		}
		if cpsTestOn {
			cpsStr = ui.CpsToString(cps)
		}
		if ppsTestOn {
			ppsStr = ui.PpsToString(pps)
		}
		if latTestOn {
			latStr = ui.DurationToString(lat.Avg)
		}
		return []string{
			ui.TruncateStringFromStart(test.RemoteIP.String(), 13),
			protocol.String(),
			bwStr,
			cpsStr,
			ppsStr,
			latStr,
		}
	}

	return []string{}
}

func (t *Tui) AddInfoMsg(msg string) {
	t.ringLock.Lock()
	parts := ui.SplitString(msg, t.msgW)
	t.msgRing = t.msgRing[len(parts):]
	t.msgRing = append(t.msgRing, parts...)
	t.ringLock.Unlock()
}

func (t *Tui) AddErrorMsg(msg string) {
	parts := ui.SplitString(msg, t.errW)
	t.ringLock.Lock()
	t.errRing = t.errRing[len(parts):]
	t.errRing = append(t.errRing, parts...)
	t.ringLock.Unlock()
}
