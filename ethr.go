//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

const defaultLogFileName = "./ethrs.log for server, ./ethrc.log for client"
const latencyDefaultBufferLenStr = "1B"
const defaultBufferLenStr = "16KB"

var (
	gVersion     string
	loggingLevel LogLevel = LogLevelInfo
)

func main() {
	//
	// If version is not set via ldflags, then default to UNKNOWN
	//
	if gVersion == "" {
		gVersion = "UNKNOWN"
	}

	fmt.Println("\nEthr: Comprehensive Network Performance Measurement Tool (Version: " + gVersion + ")")
	fmt.Println("Maintainer: Pankaj Garg (ipankajg @ LinkedIn | GitHub | Gmail | Twitter)")
	fmt.Println("")

	//
	// Set GOMAXPROCS to 1024 as running large number of goroutines that send
	// data in a tight loop over network is resulting in unfair time allocation
	// across goroutines causing starvation of many TCP connections. Using a
	// higher number of threads via GOMAXPROCS solves this problem.
	//
	runtime.GOMAXPROCS(1024)

	// Common
	flag.Usage = func() { ethrUsage() }
	noOutput := flag.Bool("no", false, "")
	outputFile := flag.String("o", defaultLogFileName, "")
	debug := flag.Bool("debug", false, "")
	use4 := flag.Bool("4", false, "")
	use6 := flag.Bool("6", false, "")
	port := flag.Int("port", 8888, "")
	ip := flag.String("ip", "", "")
	// Server
	isServer := flag.Bool("s", false, "")
	showUI := flag.Bool("ui", false, "")
	// Client & External Client
	clientDest := flag.String("c", "", "")
	bufLenStr := flag.String("l", "", "")
	bwRateStr := flag.String("b", "", "")
	cport := flag.Int("cport", 0, "")
	duration := flag.Duration("d", 10*time.Second, "")
	gap := flag.Duration("g", time.Second, "")
	iterCount := flag.Int("i", 1000, "")
	ncs := flag.Bool("ncs", false, "")
	protocol := flag.String("p", "tcp", "")
	reverse := flag.Bool("r", false, "")
	testTypePtr := flag.String("t", "", "")
	tos := flag.Int("tos", 0, "")
	title := flag.String("T", "", "")
	thCount := flag.Int("n", 1, "")
	wc := flag.Int("w", 1, "")
	xClientDest := flag.String("x", "", "")

	flag.Parse()

	if *isServer {
		if *clientDest != "" {
			printUsageError("Invalid arguments, \"-c\" cannot be used with \"-s\".")
		}
		if *xClientDest != "" {
			printUsageError("Invalid arguments, \"-x\" cannot be used with \"-s\".")
		}
		if *bufLenStr != "" {
			printServerModeArgError("l")
		}
		if *bwRateStr != "" {
			printServerModeArgError("b")
		}
		if *cport != 0 {
			printServerModeArgError("cport")
		}
		if *duration != 10*time.Second {
			printServerModeArgError("d")
		}
		if *gap != time.Second {
			printServerModeArgError("g")
		}
		if *iterCount != 1000 {
			printServerModeArgError("i")
		}
		if *ncs {
			printServerModeArgError("ncs")
		}
		if *protocol != "tcp" {
			printServerModeArgError("p")
		}
		if *reverse {
			printServerModeArgError("r")
		}
		if *testTypePtr != "" {
			printServerModeArgError("t")
		}
		if *tos != 0 {
			printServerModeArgError("tos")
		}
		if *thCount != 1 {
			printServerModeArgError("n")
		}
		if *wc != 1 {
			printServerModeArgError("wc")
		}
		if *title != "" {
			printServerModeArgError("T")
		}
	} else if *clientDest != "" || *xClientDest != "" {
		if *clientDest != "" && *xClientDest != "" {
			printUsageError("Invalid argument, both \"-c\" and \"-x\" cannot be specified at the same time.")
		}
		if *showUI {
			printUsageError(fmt.Sprintf("Invalid argument, \"-%s\" can only be used in server (\"-s\") mode.", "ui"))
		}
	} else {
		printUsageError("Invalid arguments, use either \"-s\" or \"-c\".")
	}

	// Process common parameters.

	if *debug {
		loggingLevel = LogLevelDebug
	}

	if *use4 && !*use6 {
		gIPVersion = ethrIPv4
	} else if *use6 && !*use4 {
		gIPVersion = ethrIPv6
	}

	if *ip != "" {
		gLocalIP = *ip
		ipAddr := net.ParseIP(gLocalIP)
		if ipAddr == nil {
			printUsageError(fmt.Sprintf("Invalid IP address: <%s> specified.", *ip))
		}
		if (gIPVersion == ethrIPv4 && ipAddr.To4() == nil) || (gIPVersion == ethrIPv6 && ipAddr.To16() == nil) {
			printUsageError(fmt.Sprintf("Invalid IP address version: <%s> specified.", *ip))
		}
	}
	gEthrPort = uint16(*port)
	gEthrPortStr = fmt.Sprintf("%d", gEthrPort)

	logFileName := *outputFile
	if !*noOutput {
		if logFileName == defaultLogFileName {
			if *isServer {
				logFileName = "ethrs.log"
			} else {
				logFileName = "ethrc.log"
			}
		}
		logInit(logFileName)
	}

	var testType EthrTestType
	var destination string
	if *isServer {
		// Server side parameter processing.
		testType = All
		serverParam := ethrServerParam{*showUI}
		runServer(serverParam)
	} else {
		gIsExternalClient = false
		destination = *clientDest
		if *xClientDest != "" {
			gIsExternalClient = true
			destination = *xClientDest
		}
		gNoConnectionStats = *ncs
		testType = getTestType(*testTypePtr)
		proto := getProtocol(*protocol)

		// Default latency test to 1B if length is not specified
		switch *bufLenStr {
		case "":
			*bufLenStr = getDefaultBufferLenStr(*testTypePtr)
		}
		bufLen := unitToNumber(*bufLenStr)
		if bufLen == 0 {
			printUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
		}

		// Check specific bwRate if any.
		bwRate := uint64(0)
		if *bwRateStr != "" {
			bwRate = unitToNumber(*bwRateStr)
			bwRate /= 8
		}

		//
		// For Pkt/s, we always override the buffer size to be just 1 byte.
		// TODO: Evaluate in future, if we need to support > 1 byte packets for
		//       Pkt/s testing.
		//
		if testType == Pps {
			bufLen = 1
		}

		if *iterCount <= 0 {
			printUsageError(fmt.Sprintf("Invalid iteration count for latency test: %d", *iterCount))
		}

		if *thCount <= 0 {
			*thCount = runtime.NumCPU()
		}

		gClientPort = uint16(*cport)

		testId := EthrTestID{EthrProtocol(proto), testType}
		clientParam := EthrClientParam{
			uint32(*thCount),
			uint32(bufLen),
			uint32(*iterCount),
			*reverse,
			*duration,
			*gap,
			uint32(*wc),
			uint64(bwRate),
			uint8(*tos)}
		validateClientParams(testId, clientParam)

		rServer := destination
		runClient(testId, *title, clientParam, rServer)
	}
}

func getProtocol(protoStr string) (proto EthrProtocol) {
	p := strings.ToUpper(protoStr)
	proto = TCP
	switch p {
	case "TCP":
		proto = TCP
	case "UDP":
		proto = UDP
	case "ICMP":
		proto = ICMP
	default:
		printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", protoStr))
	}
	return
}

func getTestType(testTypeStr string) (testType EthrTestType) {
	switch testTypeStr {
	case "":
		if gIsExternalClient {
			testType = Ping
		} else {
			testType = Bandwidth
		}
	case "b":
		testType = Bandwidth
	case "c":
		testType = Cps
	case "p":
		testType = Pps
	case "l":
		testType = Latency
	case "pi":
		testType = Ping
	case "tr":
		testType = TraceRoute
	case "mtr":
		testType = MyTraceRoute
	default:
		printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
			"Valid parameters and values are:\n", testTypeStr))
	}
	return
}

func getDefaultBufferLenStr(testTypePtr string) string {
	if testTypePtr == "l" {
		return latencyDefaultBufferLenStr
	}
	return defaultBufferLenStr
}

func validateClientParams(testID EthrTestID, clientParam EthrClientParam) {
	if !gIsExternalClient {
		validateClientTest(testID, clientParam)
	} else {
		validateExtModeClientTest(testID)
	}
}

func validateClientTest(testID EthrTestID, clientParam EthrClientParam) {
	testType := testID.Type
	protocol := testID.Protocol
	switch protocol {
	case TCP:
		if testType != Bandwidth && testType != Cps && testType != Latency && testType != Ping && testType != TraceRoute && testType != MyTraceRoute {
			emitUnsupportedTest(testID)
		}
		if clientParam.Reverse && testType != Bandwidth {
			printReverseModeError()
		}
		if clientParam.BufferSize > 2*GIGA {
			printUsageError("Maximum allowed value for \"-l\" for TCP is 2GB.")
		}
	case UDP:
		if testType != Bandwidth && testType != Pps {
			emitUnsupportedTest(testID)
		}
		if testType == Bandwidth {
			if clientParam.BufferSize > (64 * 1024) {
				printUsageError("Maximum supported buffer size for UDP is 64K\n")
			}
		}
		if clientParam.Reverse {
			printReverseModeError()
		}
		if clientParam.BufferSize > 64*KILO {
			printUsageError("Maximum allowed value for \"-l\" for TCP is 64KB.")
		}
	default:
		emitUnsupportedTest(testID)
	}
}

func validateExtModeClientTest(testID EthrTestID) {
	testType := testID.Type
	protocol := testID.Protocol
	switch protocol {
	case TCP:
		if testType != Ping && testType != Cps && testType != TraceRoute && testType != MyTraceRoute {
			emitUnsupportedTest(testID)
		}
	case ICMP:
		if testType != Ping && testType != TraceRoute && testType != MyTraceRoute {
			emitUnsupportedTest(testID)
		}
	default:
		emitUnsupportedTest(testID)
	}
}

func printServerModeArgError(arg string) {
	printUsageError(fmt.Sprintf("Invalid argument, \"-%s\" can only be used in client (\"-c\") mode.", arg))
}

func emitUnsupportedTest(testID EthrTestID) {
	printUsageError(fmt.Sprintf("Test: \"%s\" for Protocol: \"%s\" is not supported.\n",
		testToString(testID.Type), protoToString(testID.Protocol)))
}

func printReverseModeError() {
	printUsageError("Reverse mode (-r) is only supported for TCP Bandwidth tests.")
}

func printUsageError(s string) {
	fmt.Printf("Error: %s\n", s)
	fmt.Printf("Please use \"ethr -h\" for complete list of command line arguments.\n")
	os.Exit(1)
}

// ethrUsage prints the command-line usage text
func ethrUsage() {
	fmt.Println("Ethr supports three modes. Usage of each mode is described below:")

	fmt.Println("\nCommon Parameters")
	fmt.Println("================================================================================")
	printFlagUsage("h", "", "Help")
	printFlagUsage("no", "", "Disable logging to file. Logging to file is enabled by default.")
	printFlagUsage("o", "<filename>", "Name of log file. By default, following file names are used:",
		"Server mode: 'ethrs.log'",
		"Client mode: 'ethrc.log'")
	printFlagUsage("debug", "", "Enable debug information in logging output.")
	printFlagUsage("4", "", "Use only IP v4 version")
	printFlagUsage("6", "", "Use only IP v6 version")

	fmt.Println("\nMode: Server")
	fmt.Println("================================================================================")
	fmt.Println("In this mode, Ethr runs as a server, allowing multiple clients to run")
	fmt.Println("performance tests against it.")
	printServerUsage()
	printIPUsage()
	printPortUsage()
	printFlagUsage("ui", "", "Show output in text UI.")

	fmt.Println("\nMode: Client")
	fmt.Println("================================================================================")
	fmt.Println("In this mode, Ethr client can only talk to an Ethr server.")
	printClientUsage()
	printBwRateUsage()
	printCPortUsage()
	printDurationUsage()
	printGapUsage()
	printIterationUsage()
	printIPUsage()
	printBufLenUsage()
	printThreadUsage()
	printProtocolUsage()
	printPortUsage()
	printFlagUsage("r", "", "For Bandwidth tests, send data from server to client.")
	printTestType()
	printToSUsage()
	printWarmupUsage()
	printTitleUsage()

	fmt.Println("\nMode: External")
	fmt.Println("================================================================================")
	fmt.Println("In this mode, Ethr talks to a non-Ethr server. This mode supports only a")
	fmt.Println("few types of measurements, such as Ping, Connections/s and TraceRoute.")
	printExtClientUsage()
	printCPortUsage()
	printDurationUsage()
	printGapUsage()
	printIPUsage()
	printThreadUsage()
	printExtProtocolUsage()
	printExtTestType()
	printToSUsage()
	printWarmupUsage()
	printTitleUsage()
}

func printFlagUsage(flag, info string, helptext ...string) {
	fmt.Printf("\t-%s %s\n", flag, info)
	for _, help := range helptext {
		fmt.Printf("\t\t%s\n", help)
	}
}

func printServerUsage() {
	printFlagUsage("s", "", "Run in server mode.")
}

func printClientUsage() {
	printFlagUsage("c", "<server>", "Run in client mode and connect to <server>.",
		"Server is specified using name, FQDN or IP address.")
}

func printExtClientUsage() {
	printFlagUsage("x", "<destination>", "Run in external client mode and connect to <destination>.",
		"<destination> is specified in URL or Host:Port format.",
		"For URL, if port is not specified, it is assumed to be 80 for http and 443 for https.",
		"Example: For TCP - www.microsoft.com:443 or 10.1.0.4:22 or https://www.github.com",
		"         For ICMP - www.microsoft.com or 10.1.0.4")
}

func printPortUsage() {
	printFlagUsage("port", "<number>", "Use specified port number for TCP & UDP tests.",
		"Default: 8888")
}

func printTestType() {
	printFlagUsage("t", "<test>", "Test to run (\"b\", \"c\", \"p\", \"l\", \"cl\" or \"tr\")",
		"b: Bandwidth",
		"c: Connections/s",
		"p: Packets/s",
		"l: Latency, Loss & Jitter",
		"pi: Ping Loss & Latency",
		"tr: TraceRoute",
		"mtr: MyTraceRoute with Loss & Latency",
		"Default: b - Bandwidth measurement.")
}

func printExtTestType() {
	printFlagUsage("t", "<test>", "Test to run (\"c\", \"cl\", or \"tr\")",
		"c: Connections/s",
		"pi: Ping Loss & Latency",
		"tr: TraceRoute",
		"mtr: MyTraceRoute with Loss & Latency",
		"Default: pi - Ping Loss & Latency.")
}

func printThreadUsage() {
	printFlagUsage("n", "<number>", "Number of Parallel Sessions (and Threads).",
		"0: Equal to number of CPUs",
		"Default: 1")
}

func printDurationUsage() {
	printFlagUsage("d", "<duration>",
		"Duration for the test (format: <num>[ms | s | m | h]",
		"0: Run forever",
		"Default: 10s")
}

func printGapUsage() {
	printFlagUsage("g", "<gap>",
		"Time interval between successive measurements (format: <num>[ms | s | m | h]",
		"Only valid for latency, ping and traceRoute tests.",
		"0: No gap",
		"Default: 1s")
}

func printBufLenUsage() {
	printFlagUsage("l", "<length>",
		"Length of buffer (in Bytes) to use (format: <num>[KB | MB | GB])",
		"Only valid for Bandwidth tests. Max 1GB.",
		"Default: 16KB")
}

func printProtocolUsage() {
	printFlagUsage("p", "<protocol>",
		"Protocol (\"tcp\", \"udp\", \"http\", \"https\", or \"icmp\")",
		"Default: tcp")
}

func printExtProtocolUsage() {
	printFlagUsage("p", "<protocol>",
		"Protocol (\"tcp\", or \"icmp\")",
		"Default: tcp")
}

func printIterationUsage() {
	printFlagUsage("i", "<iterations>",
		"Number of round trip iterations for each latency measurement.",
		"Only valid for latency testing.",
		"Default: 1000")
}

func printNoConnStatUsage() {
	printFlagUsage("ncs", "",
		"No per Connection Stats would be printed if this flag is specified.",
		"This is useful to suppress verbose logging when large number of",
		"connections are used as specified by -n option for Bandwidth tests.")
}

func printIgnoreCertUsage() {
	printFlagUsage("ic", "",
		"Ignore Certificate is useful for HTTPS tests, for cases where a",
		"middle box like a proxy is not able to supply a valid Ethr cert.")
}

func printWarmupUsage() {
	printFlagUsage("w", "<number>", "Use specified number of iterations for warmup.",
		"Default: 1")
}

func printToSUsage() {
	printFlagUsage("tos", "",
		"Specifies 8-bit value to use in IPv4 TOS field or IPv6 Traffic Class field.")
}

func printBwRateUsage() {
	printFlagUsage("b", "<rate>",
		"Transmit only Bits per second (format: <num>[K | M | G])",
		"Only valid for Bandwidth tests. Default: 0 - Unlimited",
		"Examples: 100 (100bits/s), 1M (1Mbits/s).")
}

func printCPortUsage() {
	printFlagUsage("cport", "<number>", "Use specified local port number in client for TCP & UDP tests.",
		"Default: 0 - Ephemeral Port")
}

func printIPUsage() {
	printFlagUsage("ip", "<string>", "Bind to specified local IP address for TCP & UDP tests.",
		"This must be a valid IPv4 or IPv6 address.",
		"Default: <empty> - Any IP")
}

func printTitleUsage() {
	printFlagUsage("T", "<string>",
		"Use the given title in log files for logging results.",
		"Default: <empty>")
}
