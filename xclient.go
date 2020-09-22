//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"
)

func runXClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	initClient()
	test, err := newTest(server, nil, testParam, nil, nil)
	if err != nil {
		ui.printErr("runXClient: failed to create the new test.")
		return
	}
	xcRunTest(test, clientParam.duration, clientParam.gap)
}

func initXClient() {
	initClientUI()
}

func xcRunTest(test *ethrTest, d, g time.Duration) {
	startStatsTimer()
	if test.testParam.TestID.Protocol == TCP {
		if test.testParam.TestID.Type == ConnLatency {
			go xcRunTCPConnLatencyTest(test, g)
		} else if test.testParam.TestID.Type == Bandwidth {
			go xcRunTCPBandwidthTest(test)
		} else if test.testParam.TestID.Type == Cps {
			go xcRunTCPCpsTest(test)
		}
	}
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	handleInterrupt(toStop)
	reason := <-toStop
	close(test.done)
	stopStatsTimer()
	switch reason {
	case timeout:
		ui.printMsg("Ethr done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Ethr done, received interrupt signal.")
	}
}

func xcRunTCPConnLatencyTest(test *ethrTest, g time.Duration) {
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
						ui.printDbg("Unable to dial TCP connection to [%s], error: %v", server, err)
						ui.printMsg("[tcp] %sConnection %s: Timed out", warmupText, server)
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
					if t1 < g {
						time.Sleep(g - t1)
					}
				}
			}
		}()
	}
}

func xcRunTCPCpsTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					conn, err := net.Dial(tcp(ipVer), server)
					if err == nil {
						atomic.AddUint64(&test.testResult.data, 1)
						tcpconn, ok := conn.(*net.TCPConn)
						if ok {
							tcpconn.SetLinger(0)
						}
						conn.Close()
					}
				}
			}
		}()
	}
}

func xcCloseConn(conn net.Conn, test *ethrTest) {
	test.delConn(conn)
	err := conn.Close()
	if err != nil {
		ui.printDbg("Failed to close TCP connection, error: %v", err)
	}
	xcDeleteTest(test)
}

func xcDeleteTest(test *ethrTest) {
	if test != nil {
		if safeDeleteTest(test) {
			ui.printMsg("Ethr done, server terminated the session.")
			os.Exit(0)
		}
	}
}

func xcRunTCPBandwidthTest(test *ethrTest) {
	server := test.session.remoteAddr
	ui.printMsg("Connecting to host %s, port %s", server, tcpBandwidthPort)
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = byte(i)
		}
		go func() {
			conn, err := net.Dial(tcp(ipVer), server)
			if err != nil {
				ui.printErr("xcRunTCPBandwidthTest: error in dialing TCP connection: %v", err)
				os.Exit(1)
				return
			}
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
			addRef(test)
			defer xcCloseConn(conn, test)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						// ui.printErr(err)
						// test.ctrlConn.Close()
						return
					}
					if n < blen {
						// ui.printErr("Partial write: " + strconv.Itoa(n))
						// test.ctrlConn.Close()
						return
					}
					atomic.AddUint64(&ec.data, uint64(blen))
					atomic.AddUint64(&test.testResult.data, uint64(blen))
				}
			}
		}()
	}
}
