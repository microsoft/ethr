//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"bytes"
	"encoding/gob"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"time"
)

func runClient(testParam EthrTestParam, server string, d time.Duration) {
	initClient()
	test, err := establishSession(testParam, server)
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	runTest(test, d)
}

func initClient() {
	initClientUI()
}

func establishSession(testParam EthrTestParam, server string) (test *ethrTest, err error) {
	conn, err := net.Dial(protoTCP, server+":"+ctrlPort)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	ethrMsg := createSynMsg(testParam)
	err = sendSessionMsg(enc, ethrMsg)
	if err != nil {
		return
	}
	test, err = newTest(server, conn, testParam, enc, dec)
	if err != nil {
		ethrMsg = createFinMsg(err.Error())
		sendSessionMsg(enc, ethrMsg)
		return
	}
	// TODO: Enable this in future, right now there is not much value coming
	// from this.
	/**
		ethrMsg = recvSessionMsg(test.dec)
		if ethrMsg.Type != EthrAck {
			if ethrMsg.Type == EthrFin {
				err = fmt.Errorf("%s", ethrMsg.Fin.Message)
			} else {
				err = fmt.Errorf("Unexpected control message received. %v", ethrMsg)
			}
			deleteTest(test)
		}
		ethrMsg = createAckMsg()
		err = sendSessionMsg(test.enc, ethrMsg)
		if err != nil {
			os.Exit(1)
		}
	    **/
	return
}

const (
	timeout    = 0
	interrupt  = 1
	serverDone = 2
)

func handleCtrlC(toStop chan int) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		switch sig {
		case os.Interrupt:
			fallthrough
		case os.Kill:
			toStop <- interrupt
		}
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

func clientWatchControlChannel(test *ethrTest, toStop chan int) {
	go func() {
		waitForChannelStop := make(chan bool, 1)
		watchControlChannel(test, waitForChannelStop)
		<-waitForChannelStop
		toStop <- serverDone
	}()
}

func runTest(test *ethrTest, d time.Duration) {
	startStatsTimer()
	if test.testParam.TestID.Protocol == TCP {
		if test.testParam.TestID.Type == Bandwidth {
			go runTCPBandwidthTest(test)
		} else if test.testParam.TestID.Type == Cps {
			go runTCPCpsTest(test)
		} else if test.testParam.TestID.Type == Latency {
			ui.emitLatencyHdr()
			go runTCPLatencyTest(test)
		}
	} else if test.testParam.TestID.Protocol == UDP {
		if test.testParam.TestID.Type == Bandwidth {
			go runUDPBandwidthTest(test)
		} else if test.testParam.TestID.Type == Pps {
			go runUDPPpsTest(test)
		}
	} else if test.testParam.TestID.Protocol == HTTP {
		if test.testParam.TestID.Type == Bandwidth {
			go runHTTPBandwidthTest(test)
		}
	}
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	clientWatchControlChannel(test, toStop)
	handleCtrlC(toStop)
	reason := <-toStop
	close(test.done)
	test.ctrlConn.Close()
	stopStatsTimer()
	switch reason {
	case timeout:
		ui.printMsg("Ethr done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Ethr done, received interrupt signal.")
	case serverDone:
		ui.printMsg("Ethr done, server terminated the session.")
	}
}

func runTCPBandwidthTest(test *ethrTest) {
	server := test.session.remoteAddr
	ui.printMsg("Connecting to host %s, port %s", server, tcpBandwidthPort)
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = byte(i)
		}
		go func() {
			conn, err := net.Dial(protoTCP, server+":"+tcpBandwidthPort)
			if err != nil {
				ui.printErr("%v", err)
				os.Exit(1)
				return
			}
			defer conn.Close()
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
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
						// return
						continue
					}
					if n < blen {
						// ui.printErr("Partial write: " + strconv.Itoa(n))
						// test.ctrlConn.Close()
						// return
						continue
					}
					atomic.AddUint64(&ec.data, uint64(blen))
					atomic.AddUint64(&test.testResult.data, uint64(blen))
				}
			}
		}()
	}
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
					conn, err := net.Dial(protoTCP, server+":"+tcpCpsPort)
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

func runUDPBandwidthTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial(protoUDP, server+":"+udpBandwidthPort)
			if err != nil {
				ui.printDbg("Unable to dial UDP, error: %v", err)
				return
			}
			defer conn.Close()
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						ui.printDbg("%v", err)
						continue
					}
					if n < blen {
						ui.printDbg("Partial write: %d", n)
						continue
					}
					atomic.AddUint64(&ec.data, uint64(n))
					atomic.AddUint64(&test.testResult.data, uint64(n))
				}
			}
		}()
	}
}

func runUDPPpsTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial(protoUDP, server+":"+udpPpsPort)
			if err != nil {
				ui.printDbg("Unable to dial UDP, error: %v", err)
				return
			}
			defer conn.Close()
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[udp] local %s port %s connected to %s port %s",
				lserver, lport, rserver, rport)
			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						ui.printDbg("%v", err)
						continue
					}
					if n < blen {
						ui.printDbg("Partial write: %d", n)
						continue
					}
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}

func runTCPLatencyTest(test *ethrTest) {
	server := test.session.remoteAddr
	conn, err := net.Dial(protoTCP, server+":"+tcpLatencyPort)
	if err != nil {
		ui.printErr("Error dialing the latency connection: %v", err)
		os.Exit(1)
		return
	}
	defer conn.Close()
	buffSize := test.testParam.BufferSize
	// TODO Override buffer size to 1 for now. Evaluate if we need to allow
	// client to specify the buffer size in future.
	buffSize = 1
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
				if err != nil {
					// ui.printErr(err)
					// return
					break ExitSelect
				}
				if n < blen {
					// ui.printErr("Partial write: " + strconv.Itoa(n))
					// return
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					// ui.printErr(err)
					// return
					break ExitSelect
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}
			// TODO temp code, fix it better, this is to allow server to do
			// server side latency measurements as well.
			_, _ = conn.Write(buff)
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
	}
}

func runHTTPBandwidthTest(test *ethrTest) {
	uri := test.session.remoteAddr
	uri = "http://" + uri + ":" + httpBandwidthPort
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			// buff[i] = byte(i)
			buff[i] = 'x'
		}
		tr := &http.Transport{DisableCompression: true}
		client := &http.Client{Transport: tr}
		go func() {
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					// response, err := http.Get(uri)
					response, err := client.Post(uri, "text/plain", bytes.NewBuffer(buff))
					if err != nil {
						// ui.printErr("%v", err)
						continue
					} else {
						if response.StatusCode != http.StatusOK {
							continue
						}
						// contents, err := ioutil.ReadAll(response.Body)
						_, err = ioutil.ReadAll(response.Body)
						response.Body.Close()
						if err != nil {
							// ui.printErr("%v", err)
							continue
						}
						// ui.printMsg("%s", string(contents))
					}
					atomic.AddUint64(&test.testResult.data, uint64(test.testParam.BufferSize))
				}
			}
		}()
	}
}
