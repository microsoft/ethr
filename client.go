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

	"bytes"
	"encoding/binary"
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
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var gIgnoreCert bool

const (
	timeout    = 0
	interrupt  = 1
	disconnect = 2
)

func handleInterrupt(toStop chan<- int) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
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
		// Sleep extra 200ms to ensure stats print for correct number of seconds.
		time.Sleep(200 * time.Millisecond)
		toStop <- timeout
	}()
}

func initClient() {
	initClientUI()
}

func handshakeWithServer(test *ethrTest, conn net.Conn) {
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

func runClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	initClient()
	if !xMode {
		addr := net.ParseIP(server)
		if addr != nil {
			// TODO - PG -
			//server = "[" + server + "]"
		}
	}
	test, err := newTest(server, nil, testParam, nil, nil)
	if err != nil {
		ui.printErr("Failed to create the new test.")
		return
	}
	runTest(test, clientParam.duration, clientParam.gap)
}

func runTest(test *ethrTest, d, g time.Duration) {
	toStop := make(chan int, 1)
	startStatsTimer()
	runDurationTimer(d, toStop)
	test.isActive = true
	/*
		if test.testParam.TestID.Protocol == TCP {
			if test.testParam.TestID.Type == Bandwidth {
				runTCPBandwidthTest(test, toStop)
			} else if test.testParam.TestID.Type == Latency {
				go runTCPLatencyTest(test, g, toStop)
			} else if test.testParam.TestID.Type == Cps {
				go runTCPCpsTest(test)
			} else if test.testParam.TestID.Type == ConnLatency {
				go runTCPConnLatencyTest(test, g)
			}
		} else if test.testParam.TestID.Protocol == UDP {
			if test.testParam.TestID.Type == Bandwidth ||
				test.testParam.TestID.Type == Pps {
				runUDPBandwidthAndPpsTest(test)
			}
		}
	*/
	runICMPTraceRoute(test)
	handleInterrupt(toStop)
	reason := <-toStop
	stopStatsTimer()
	close(test.done)
	if test.testParam.TestID.Type == ConnLatency {
		time.Sleep(2 * time.Second)
	}
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

func runTCPBandwidthTest(test *ethrTest, toStop chan int) {
	var wg sync.WaitGroup
	runTCPBandwidthTestThreads(test, &wg)
	go func(wg *sync.WaitGroup) {
		wg.Wait()
		toStop <- disconnect
	}(&wg)
}

func runTCPBandwidthTestThreads(test *ethrTest, wg *sync.WaitGroup) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
		if err != nil {
			ui.printErr("Error dialing connection: %v", err)
			return
		}
		handshakeWithServer(test, conn)
		wg.Add(1)
		go runTCPBandwidthTestHandler(test, conn, wg)
	}
}

func runTCPBandwidthTestHandler(test *ethrTest, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()
	ec := test.newConn(conn)
	rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
	lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
	ui.printMsg("[%3d] local %s port %s connected to %s port %s",
		ec.fd, lserver, lport, rserver, rport)
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
			atomic.AddUint64(&ec.bw, uint64(blen))
			atomic.AddUint64(&test.testResult.bw, uint64(blen))
		}
	}
}

func runTCPLatencyTest(test *ethrTest, g time.Duration, toStop chan int) {
	server := test.session.remoteAddr
	conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
	if err != nil {
		ui.printErr("Error dialing the latency connection: %v", err)
		os.Exit(1)
		return
	}
	defer conn.Close()
	handshakeWithServer(test, conn)
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
			t0 := time.Now()
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
			calcAndPrintLatency(test, rttCount, latencyNumbers)
			t1 := time.Since(t0)
			if t1 < g {
				time.Sleep(g - t1)
			}
		}
	}
}

func calcAndPrintLatency(test *ethrTest, rttCount uint32, latencyNumbers []time.Duration) {
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
					rserver := server
					if !xMode {
						rserver = rserver + ":" + ctrlPort
					}
					conn, err := net.Dial(tcp(ipVer), rserver)
					if err == nil {
						atomic.AddUint64(&test.testResult.cps, 1)
						tcpconn, ok := conn.(*net.TCPConn)
						if ok {
							tcpconn.SetLinger(0)
						}
						conn.Close()
					} else {
						ui.printDbg("Unable to dial TCP connection to [%s], error: %v", rserver, err)
					}
				}
			}
		}()
	}
}

func runTCPConnLatencyTest(test *ethrTest, g time.Duration) {
	server := test.session.remoteAddr
	if !xMode {
		server = server + ":" + ctrlPort
	}
	// TODO: Override NumThreads for now, fix it later to support parallel
	// threads.
	test.testParam.NumThreads = 1
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			var sent, rcvd, lost uint32
			var warmupCount int32 = 1
			warmupText := "[warmup] "
			latencyNumbers := make([]time.Duration, 0)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					printConnectionLatencyResults(server, test, sent, rcvd, lost, latencyNumbers)
					break ExitForLoop
				default:
					t0 := time.Now()
					if warmupCount > 0 {
						warmupCount--
						dialForConnectionLatency(&server, warmupText)
					} else {
						sent++
						latency, err := dialForConnectionLatency(&server, "")
						if err == nil {
							rcvd++
							latencyNumbers = append(latencyNumbers, latency)
						} else {
							lost++
						}
					}
					if rcvd >= 1000 {
						printConnectionLatencyResults(server, test, sent, rcvd, lost, latencyNumbers)
						latencyNumbers = make([]time.Duration, 0)
						sent, rcvd, lost = 0, 0, 0
					}
					t1 := time.Since(t0)
					if t1 < g {
						time.Sleep(g - t1)
					}
				}
			}
		}()
	}
}

func dialForConnectionLatency(server *string, prefix string) (timeTaken time.Duration, err error) {
	t0 := time.Now()
	// conn, err := net.DialTimeout(tcp(ipVer), *server, time.Second)
	conn, err := net.Dial(tcp(ipVer), *server)
	if err != nil {
		ui.printDbg("Unable to dial TCP connection to [%s], error: %v", *server, err)
		ui.printMsg("[tcp] %sConnection to %s: Timed out (%v)", prefix, *server, err)
		return
	}
	timeTaken = time.Since(t0)
	rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
	lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
	ui.printMsg("[tcp] %sConnection from [%s]:%s to [%s]:%s: %s",
		prefix, lserver, lport, rserver, rport, durationToString(timeTaken))
	*server = fmt.Sprintf("[%s]:%s", rserver, rport)
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		tcpconn.SetLinger(0)
	}
	conn.Close()
	return
}

func printConnectionLatencyResults(server string, test *ethrTest, sent, rcvd, lost uint32, latencyNumbers []time.Duration) {
	fmt.Println("-----------------------------------------------------------------------------------------")
	ui.printMsg("TCP connect statistics for %s:", server)
	ui.printMsg("  Sent = %d, Received = %d, Lost = %d", sent, rcvd, lost)
	if rcvd > 0 {
		ui.emitLatencyHdr()
		calcAndPrintLatency(test, rcvd, latencyNumbers)
		fmt.Println("-----------------------------------------------------------------------------------------")
	}
}

func runUDPBandwidthAndPpsTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial(udp(ipVer), server+":"+ctrlPort)
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
					atomic.AddUint64(&ec.bw, uint64(n))
					atomic.AddUint64(&ec.pps, 1)
					atomic.AddUint64(&test.testResult.bw, uint64(n))
					atomic.AddUint64(&test.testResult.pps, 1)
				}
			}
		}()
	}
}

type ethrHopData struct {
	addr  net.Addr
	sent  uint32
	rcvd  uint32
	lost  uint32
	last  time.Duration
	best  time.Duration
	worst time.Duration
	total time.Duration
	name  string
}

var gMaxHops int = 8
var gHop [64]ethrHopData

func runICMPTraceRoute(test *ethrTest) {
	for i := 0; i < gMaxHops; i++ {
		go runICMPTraceRouteInternal(test, i)
	}

ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			printICMPHopData()
		}
	}
}

func printICMPHopData() {
	for i := 0; i < gMaxHops; i++ {
		hopData := gHop[i]
		if hopData.addr != nil {
			ui.printMsg("Hop: %v, Address: %v, Name: %v, Sent: %v, Rcvd: %v, Last: %v", i, hopData.addr, hopData.name, hopData.sent, hopData.rcvd, hopData.last)
		}
	}
	time.Sleep(9 * time.Second)
}

func runICMPTraceRouteInternal(test *ethrTest, hop int) {
	seq := 0
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			sendICMPEcho(test, &gHop[hop], hop, seq)
			seq++
			time.Sleep(9 * time.Second)
		}
	}
}

func sendICMPEcho(test *ethrTest, hopData *ethrHopData, hop, seq int) {
	server := test.session.remoteAddr
	peer := ""
	start := time.Now()
	localAddr := ""
	c, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		ui.printErr("Failed to listen to local address %v. Msg: %v.", localAddr, err.Error())
		return
	}
	defer c.Close()
	ui.printMsg("TTL: %v", hop+1)
	err = c.IPv4PacketConn().SetTTL(hop + 1)
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	tm := 10 * time.Second
	err = c.SetDeadline(time.Now().Add(tm))
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	pid := os.Getpid() & 0xffff
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(seq))
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: pid, Seq: seq,
			Data: append(bs, 'x'),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	destIP := server
	ui.printMsg("Server: %v", server)
	if net.ParseIP(server) == nil {
		addrs, err := net.LookupHost(server)
		if err != nil || len(addrs) == 0 {
			ui.printErr("%v", "Kela")
			return
		}
		ui.printMsg("addrs: %v", addrs)
		destIP = addrs[0]
	}
	ipAddr := net.IPAddr{IP: net.ParseIP(destIP)}
	ui.printMsg("server: %v, ipAddr: %v", server, ipAddr)
	if _, err := c.WriteTo(wb, &ipAddr); err != nil {
		ui.printErr("%v", err)
		return
	}
	hopData.sent++
	ui.printMsg("Success0")
	peerAddr, _, err := listenForSpecific4(c, time.Now().Add(tm), peer, append(bs, 'x'), seq, wb)
	if err != nil {
		hopData.lost++
		ui.printErr("Failure1: %v", err)
		return
	}
	peer = peerAddr.String()
	elapsed := time.Since(start)
	hopData.addr = peerAddr
	hopData.last = elapsed
	if hopData.best > elapsed {
		hopData.best = elapsed
	}
	if hopData.worst < elapsed {
		hopData.worst = elapsed
	}
	hopData.total += elapsed
	hopData.rcvd++
	names, err := net.LookupAddr(peer)
	if err == nil && len(names) > 0 {
		hopData.name = names[0]
	}
	ui.printMsg("Success1")
	return
}

const (
	ProtocolICMP     = 1  // ICMP for IPv4
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

func listenForSpecific4(conn *icmp.PacketConn, deadline time.Time, neededPeer string, neededBody []byte, needSeq int, sent []byte) (net.Addr, []byte, error) {
	for {
		b := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(b)
		if err != nil {
			ui.printErr("Failure0: %v", err)
			if neterr, ok := err.(*net.OpError); ok {
				return nil, []byte{}, neterr
			}
		}
		if n == 0 {
			continue
		}

		if neededPeer != "" && peer.String() != neededPeer {
			continue
		}

		x, err := icmp.ParseMessage(ProtocolICMP, b[:n])
		if err != nil {
			continue
		}

		if typ, ok := x.Type.(ipv4.ICMPType); ok && typ == ipv4.ICMPTypeTimeExceeded {
			body := x.Body.(*icmp.TimeExceeded).Data

			index := bytes.Index(body, sent[:4])
			if index > 0 {
				x, _ := icmp.ParseMessage(ProtocolICMP, body[index:])
				switch x.Body.(type) {
				case *icmp.Echo:
					seq := x.Body.(*icmp.Echo).Seq
					if seq == needSeq {
						return peer, []byte{}, nil
					}
				default:
					// ignore
				}
			}
		}

		if typ, ok := x.Type.(ipv4.ICMPType); ok && typ == ipv4.ICMPTypeEchoReply {
			b, _ := x.Body.Marshal(1)
			if string(b[4:]) != string(neededBody) {
				continue
			}
			return peer, b[4:], nil
		}
	}
}

/*
func runHTTPBandwidthTest(test *ethrTest) {
	uri := test.session.remoteAddr
	ui.printMsg("uri=%s", uri)
	uri = "http://" + uri + ":" + httpBandwidthPort
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = 'x'
		}
		tr := &http.Transport{DisableCompression: true}
		client := &http.Client{Transport: tr}
		go runHTTPandHTTPSBandwidthTest(test, client, uri, buff)
	}
}

func runHTTPSBandwidthTest(test *ethrTest) {
	uri := test.session.remoteAddr
	uri = "https://" + uri + ":" + httpsBandwidthPort
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = 'x'
		}
		c, err := x509.ParseCertificate(gCert)
		if err != nil {
			ui.printErr("runHTTPSBandwidthTest: failed to parse certificate: %v", err)
		}
		clientCertPool := x509.NewCertPool()
		clientCertPool.AddCert(c)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: gIgnoreCert,
			// Certificates: []tls.Certificate{cert},
			RootCAs: clientCertPool,
		}
		//tlsConfig.BuildNameToCertificate()
		tr := &http.Transport{DisableCompression: true, TLSClientConfig: tlsConfig}
		client := &http.Client{Transport: tr}
		go runHTTPandHTTPSBandwidthTest(test, client, uri, buff)
	}
}

func runHTTPandHTTPSBandwidthTest(test *ethrTest, client *http.Client, uri string, buff []byte) {
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			// response, err := http.Get(uri)
			response, err := client.Post(uri, "text/plain", bytes.NewBuffer(buff))
			if err != nil {
				ui.printDbg("Error in HTTP request: %v", err)
				continue
			} else {
				ui.printDbg("Status received: %v", response.StatusCode)
				if response.StatusCode != http.StatusOK {
					ui.printDbg("Error in HTTP request, received status: %v", response.StatusCode)
					continue
				}
				contents, err := ioutil.ReadAll(response.Body)
				response.Body.Close()
				if err != nil {
					ui.printDbg("Error in receving HTTP response: %v", err)
					continue
				}
				ethrUnused(contents)
				// ui.printDbg("%s", string(contents))
			}
			atomic.AddUint64(&test.testResult.data, uint64(test.testParam.BufferSize))
		}
	}
}
*/
