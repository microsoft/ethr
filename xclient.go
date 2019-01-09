//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
	"time"
)

func runXClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	initClient()
	test, err := newTest(server, nil, testParam, nil, nil)
	if err != nil {
		ui.printErr("Failed to create the new test.")
		return
	}
	xclientTest(test, clientParam.duration)
}

func initXClient() {
	initClientUI()
}

func xclientTest(test *ethrTest, d time.Duration) {
	if test.testParam.TestID.Protocol == TCP {
		if test.testParam.TestID.Type == ConnLatency {
			go xclientTCPLatencyTest(test)
		}
	}
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	handleCtrlC(toStop)
	reason := <-toStop
	switch reason {
	case timeout:
		ui.printMsg("Ethr done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Ethr done, received interrupt signal.")
	}
	ui.printMsg("")
	close(test.done)
	time.Sleep(time.Second)
}

func xclientTCPLatencyTest(test *ethrTest) {
	server := test.session.remoteAddr
	// TODO: Override NumThreads for now, fix it later to support parallel
	// threads.
	test.testParam.NumThreads = 1
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			var min, max, avg time.Duration
			var sum, sent, rcvd, lost int64
			min = time.Hour
			sent = -1
			warmup := true
			warmupText := "[warmup] "
		ExitForLoop:
			for {
				select {
				case <-test.done:
					ui.printMsg("TCP connect statistics for %s:", server)
					ui.printMsg("  Sent = %d, Received = %d, Lost = %d", sent, rcvd, lost)
					ui.printMsg("  Min = %s, Max = %s, Avg = %s", durationToString(min),
						durationToString(max), durationToString(avg))
					break ExitForLoop
				default:
					sent++
					t0 := time.Now()
					conn, err := net.Dial(tcp(ipVer), server)
					if err != nil {
						lost++
						ui.printErr("Unable to dial TCP connection to [%s], error: %v", server, err)
						continue
					}
					t1 := time.Since(t0)
					rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
					lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
					ui.printMsg("[tcp] %sConnection from [%s]:%s to [%s]:%s: %s",
						warmupText, lserver, lport, rserver, rport, durationToString(t1))
					if !warmup {
						rcvd++
						sum += t1.Nanoseconds()
						avg = time.Duration(sum / rcvd)
						if t1 < min {
							min = t1
						}
						if t1 > max {
							max = t1
						}
					} else {
						warmup = false
						warmupText = ""
						server = fmt.Sprintf("[%s]:%s", rserver, rport)
					}
					tcpconn, ok := conn.(*net.TCPConn)
					if ok {
						tcpconn.SetLinger(0)
					}
					conn.Close()

					t1 = time.Since(t0)
					if t1 < time.Second {
						// time.Sleep(time.Second - t1)
					}
				}
			}
		}()
	}
}
