//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"flag"
	"fmt"
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
	// Server
	isServer := flag.Bool("s", false, "")
	showUI := flag.Bool("ui", false, "")
	// Client
	clientDest := flag.String("c", "", "")
	bufLenStr := flag.String("l", "", "")
	duration := flag.Duration("d", 10*time.Second, "")
	gap := flag.Duration("g", time.Second, "")
	mode := flag.Bool("x", false, "")
	ncs := flag.Bool("ncs", false, "")
	protocol := flag.String("p", "tcp", "")
	reverse := flag.Bool("r", false, "")
	iterCount := flag.Int("i", 1000, "")
	testTypePtr := flag.String("t", "", "")
	thCount := flag.Int("n", 1, "")
	wc := flag.Int("w", 1, "")

	flag.Parse()

	if *isServer {
		if *clientDest != "" {
			printUsageError("Invalid arguments, \"-c\" cannot be used with \"-s\".")
		}
		if *bufLenStr != "" {
			printServerModeArgError("l")
		}
		if *duration != 10*time.Second {
			printServerModeArgError("d")
		}
		if *gap != time.Second {
			printServerModeArgError("g")
		}
		if *mode {
			printServerModeArgError("x")
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
		if *iterCount != 1000 {
			printServerModeArgError("i")
		}
		if *testTypePtr != "" {
			printServerModeArgError("t")
		}
		if *thCount != 1 {
			printServerModeArgError("n")
		}
		if *wc != 1 {
			printServerModeArgError("wc")
		}
	} else if *clientDest != "" {
		if *showUI {
			printUsageError(fmt.Sprintf("Invalid argument, \"-%s\" can only be used in server (\"-s\") mode.", arg))
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

	gEthrPort = *port
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

	if *isServer {
		// Server side parameter processing.
		testType = All
		serverParam := ethrServerParam{*showUI}
		runServer(serverParam)
	} else {
		gIsExternalClient = *mode
		gNoConnectionStats = *ncs
		switch *testTypePtr {
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
				"Valid parameters and values are:\n", *testTypePtr))
		}

		// Default latency test to 1B if length is not specified
		switch *bufLenStr {
		case "":
			*bufLenStr = getDefaultBufferLenStr(*testTypePtr)
		}
		bufLen := unitToNumber(*bufLenStr)
		if bufLen == 0 {
			printUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
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

		p := strings.ToUpper(*protocol)
		proto := TCP
		switch p {
		case "TCP":
			proto = TCP
		case "UDP":
			proto = UDP
		case "HTTP":
			proto = HTTP
		case "HTTPS":
			proto = HTTPS
		case "ICMP":
			proto = ICMP
		default:
			printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
				"Valid parameters and values are:\n", *protocol))
		}

		// Override ipVer because for ICMP, specific version is required.
		if proto == ICMP || (testType == TraceRoute || testType == MyTraceRoute) {
			if gIPVersion == ethrIPAny {
				gIPVersion = ethrIPv4
			}
		}

		if *thCount <= 0 {
			*thCount = runtime.NumCPU()
		}
		testId := EthrTestID{EthrProtocol(proto), testType}
		clientParam := EthrClientParam{
			uint32(*thCount),
			uint32(bufLen),
			uint32(*iterCount),
			*reverse,
			*duration,
			*gap,
			*wc}
		validateClientParams(clientParam)

		rServer := *clientDest
		runClient(testId, clientParam, rServer)
	}
}

func getDefaultBufferLenStr(testTypePtr string) string {
	if testTypePtr == "l" {
		return latencyDefaultBufferLenStr
	}
	return defaultBufferLenStr
}

func validateClientParams(testID EthrTestID, clientParam EthrClientParam) {
	testType := testID.Type
	protocol := testID.Protocol
	if !gIsExternalClient {
		switch protocol {
		case TCP:
			if testType != Bandwidth && testType != Cps && testType != Latency && testType != Ping && testType != TraceRoute && testType != MyTraceRoute {
				emitUnsupportedTest(testID)
			}
			if clientParam.Reverse && testType != Bandwidth {
				printReverseModeError()
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
		case ICMP:
			if testType != TraceRoute && testType != MyTraceRoute {
				emitUnsupportedTest(testID)
			}
		case HTTP:
			if testType != Bandwidth && testType != Latency {
				emitUnsupportedTest(testID)
			}
			if clientParam.Reverse {
				printReverseModeError()
			}
		case HTTPS:
			if testType != Bandwidth {
				emitUnsupportedTest(testID)
			}
			if clientParam.Reverse {
				printReverseModeError()
			}
		default:
			emitUnsupportedTest(testID)
		}
	} else {
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
	printFlagUsage("ui", "", "Show output in text UI.")
	printPortUsage()

	fmt.Println("\nMode: Client")
	fmt.Println("================================================================================")
	fmt.Println("In this mode, Ethr client can only talk to an Ethr server.")
	printClientUsage()
	printDurationUsage()
	printGapUsage()
	printIterationUsage()
	printBufLenUsage()
	printThreadUsage()
	printProtocolUsage()
	printPortUsage()
	printFlagUsage("r", "", "For Bandwidth tests, send data from server to client.")
	printTestType()
	printWarmupUsage()

	fmt.Println("\nMode: External Client")
	fmt.Println("================================================================================")
	fmt.Println("In this mode, Ethr client can talk to a non-Ethr server. This mode only supports")
	fmt.Println("few types of measurements, such as Ping, Connections/s and TraceRoute.")
	printModeUsage()
	printExtClientUsage()
	printDurationUsage()
	printGapUsage()
	printThreadUsage()
	printExtProtocolUsage()
	printExtTestType()
	printWarmupUsage()
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
	printFlagUsage("c", "<destination>", "Run in external client mode and connect to <destination>.",
		"<destination> is specified in <host:port> format for TCP and <host> format for ICMP.",
		"Example: For TCP - www.microsoft.com:443 or 10.1.0.4:22",
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
		"Length of buffer to use (format: <num>[KB | MB | GB])",
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

func printModeUsage() {
	printFlagUsage("x", "",
		"'-x' enables Ethr client to talk to non-Ethr server.")
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
