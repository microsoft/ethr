//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	//	"bytes"
	//	"crypto/tls"
	//	"crypto/x509"

	"encoding/gob"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"

	//	"io"
	//	"io/ioutil"
	"net"
	//	"net/http"
	"os"
	"os/signal"

	//	"sort"
	//	"sync/atomic"
	"time"
)

var gIgnoreCert bool

const (
	timeout    = 0
	interrupt  = 1
	disconnect = 2
)

// handleInterrupt handles os.Interrupt
// os.Interrupt guaranteed to be present on all systems
func handleInterrupt(toStop chan<- int) {
	sigChan := make(chan os.Signal)
	// TODO: Handle graceful shutdown in containers as well
	// by handling syscall.SIGTERM
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		toStop <- interrupt
	}()
}

func runDurationTimer(d time.Duration, toStop chan int) {
	go func() {
		dSeconds := uint64(d.Seconds())
		if dSeconds == 0 {
			return
		}
		time.Sleep(d)
		toStop <- timeout
	}()
}

func initClient() {
	initClientUI()
}

func runClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	initClient()
	server = "[" + server + "]"
	test, err := newTest(server, nil, testParam, nil, nil)
	if err != nil {
		ui.printErr("runXClient: failed to create the new test.")
		return
	}
	runTest(test, clientParam.duration, clientParam.gap)
}

func runTest(test *ethrTest, d, g time.Duration) {
	startStatsTimer()
	toStop := make(chan int, 1)
	if test.testParam.TestID.Protocol == TCP {
		if test.testParam.TestID.Type == Bandwidth {
			runBandwidthTest(test, toStop)
		} else if test.testParam.TestID.Type == Latency {
			go runTCPLatencyTest(test, toStop)
		} else if test.testParam.TestID.Type == Cps {
			go runTCPCpsTest(test)
		}
	}
	test.isActive = true
	runDurationTimer(d, toStop)
	handleInterrupt(toStop)
	reason := <-toStop
	stopStatsTimer()
	close(test.done)
	switch reason {
	case timeout:
		ui.printMsg("Ethr done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Ethr done, received interrupt signal.")
	case disconnect:
		ui.printMsg("Ethr done, connection terminated.")
	}
	return
}

func runBandwidthTest(test *ethrTest, toStop chan int) {
	var wg sync.WaitGroup
	runBandwidthTestThreads(test, &wg)
	go func(wg *sync.WaitGroup) {
		wg.Wait()
		toStop <- disconnect
	}(&wg)
}

func runBandwidthTestThreads(test *ethrTest, wg *sync.WaitGroup) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
		if err != nil {
			ui.printErr("Error dialing connection: %v", err)
			return
		}
		handshakeBandwidthTest(test, conn)
		wg.Add(1)
		go runTCPBandwidthTest(test, conn, wg)
	}
}

func handshakeBandwidthTest(test *ethrTest, conn net.Conn) {
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	ethrMsg := createSynMsg(test.testParam)
	err := sendSessionMsg(enc, ethrMsg)
	if err != nil {
		ui.printErr("Failed to send session message: %v", err)
		return
	}
	ethrMsg = recvSessionMsg(dec)
	if ethrMsg.Type != EthrAck {
		if ethrMsg.Type == EthrFin {
			err = fmt.Errorf("%s", ethrMsg.Fin.Message)
		} else {
			err = fmt.Errorf("Unexpected control message received. %v", ethrMsg)
		}
	}
}

func runTCPBandwidthTest(test *ethrTest, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()
	ec := test.newConn(conn)
	buff := make([]byte, test.testParam.BufferSize)
	for i := uint32(0); i < test.testParam.BufferSize; i++ {
		buff[i] = byte(i)
	}
	blen := len(buff)
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			n := 0
			var err error = nil
			if test.testParam.Reverse {
				n, err = io.ReadFull(conn, buff)
			} else {
				n, err = conn.Write(buff)
			}
			if err != nil || n < blen {
				ui.printDbg("Error sending/receiving data on a connection for bandwidth test: %v", err)
				break ExitForLoop
			}
			atomic.AddUint64(&ec.data, uint64(blen))
			atomic.AddUint64(&test.testResult.data, uint64(blen))
		}
	}
}

func runTCPLatencyTest(test *ethrTest, toStop chan int) {
	server := test.session.remoteAddr
	conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
	if err != nil {
		ui.printErr("Error dialing the latency connection: %v", err)
		os.Exit(1)
		return
	}
	defer conn.Close()
	handshakeBandwidthTest(test, conn)
	ui.emitLatencyHdr()
	buffSize := test.testParam.BufferSize
	buff := make([]byte, buffSize)
	for i := uint32(0); i < buffSize; i++ {
		buff[i] = byte(i)
	}
	blen := len(buff)
	rttCount := test.testParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
ExitForLoop:
	for {
	ExitSelect:
		select {
		case <-test.done:
			break ExitForLoop
		default:
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				n, err := conn.Write(buff)
				if err != nil || n < blen {
					ui.printDbg("Error sending/receiving data on a connection for latency test: %v", err)
					toStop <- disconnect
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					ui.printDbg("Error sending/receiving data on a connection for latency test: %v", err)
					toStop <- disconnect
					break ExitSelect
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}
			// TODO temp code, fix it better, this is to allow server to do
			// server side latency measurements as well.
			_, _ = conn.Write(buff)

			calcLatency(test, rttCount, latencyNumbers)
		}
	}
}

func calcLatency(test *ethrTest, rttCount uint32, latencyNumbers []time.Duration) {
	sum := int64(0)
	for _, d := range latencyNumbers {
		sum += d.Nanoseconds()
	}
	elapsed := time.Duration(sum / int64(rttCount))
	sort.SliceStable(latencyNumbers, func(i, j int) bool {
		return latencyNumbers[i] < latencyNumbers[j]
	})
	//
	// Special handling for rttCount == 1. This prevents negative index
	// in the latencyNumber index. The other option is to use
	// roundUpToZero() but that is more expensive.
	//
	rttCountFixed := rttCount
	if rttCountFixed == 1 {
		rttCountFixed = 2
	}
	avg := elapsed
	min := latencyNumbers[0]
	max := latencyNumbers[rttCount-1]
	p50 := latencyNumbers[((rttCountFixed*50)/100)-1]
	p90 := latencyNumbers[((rttCountFixed*90)/100)-1]
	p95 := latencyNumbers[((rttCountFixed*95)/100)-1]
	p99 := latencyNumbers[((rttCountFixed*99)/100)-1]
	p999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.9)/100)-1)]
	p9999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.99)/100)-1)]
	ui.emitLatencyResults(
		test.session.remoteAddr,
		protoToString(test.testParam.TestID.Protocol),
		avg, min, max, p50, p90, p95, p99, p999, p9999)
}

func runTCPCpsTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
					if err == nil {
						atomic.AddUint64(&test.testResult.data, 1)
						tcpconn, ok := conn.(*net.TCPConn)
						if ok {
							tcpconn.SetLinger(0)
						}
						conn.Close()
					} else {
						ui.printDbg("Error setting connection for CPS test: %v", err)
					}
				}
			}
		}()
	}
}
