//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"flag"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/microsoft/ethr/internal/cmd"
)

const defaultLogFileName = "./ethrs.log for server, ./ethrc.log for client"
const latencyDefaultBufferLenStr = "1B"
const defaultBufferLenStr = "16KB"

var gVersion string

func main() {
	//
	// If version is not set via ldflags, then default to UNKNOWN
	//
	if gVersion == "" {
		gVersion = "[VERSION: UNKNOWN]"
	}
	//
	// Set GOMAXPROCS to 1024 as running large number of goroutines in a loop
	// to send network traffic results in timer starvation, as well as unfair
	// processing time across goroutines resulting in starvation of many TCP
	// connections. Using a higher number of threads via GOMAXPROCS solves this
	// problem.
	//
	runtime.GOMAXPROCS(1024)

	flag.Usage = func() { cmd.EthrUsage(gVersion) }
	isServer := flag.Bool("s", false, "")
	clientDest := flag.String("c", "", "")
	testTypePtr := flag.String("t", "", "")
	thCount := flag.Int("n", 1, "")
	bufLenStr := flag.String("l", "", "")
	protocol := flag.String("p", "tcp", "")
	outputFile := flag.String("o", defaultLogFileName, "")
	debug := flag.Bool("debug", false, "")
	noOutput := flag.Bool("no", false, "")
	duration := flag.Duration("d", 10*time.Second, "")
	showUI := flag.Bool("ui", false, "")
	rttCount := flag.Int("i", 1000, "")
	portStr := flag.String("ports", "", "")
	modeStr := flag.String("m", "", "")
	use4 := flag.Bool("4", false, "")
	use6 := flag.Bool("6", false, "")
	gap := flag.Duration("g", 0, "")
	reverse := flag.Bool("r", false, "")
	ncs := flag.Bool("ncs", false, "")
	ic := flag.Bool("ic", false, "")

	flag.Parse()

	//
	// TODO: Handle the case if there are incorrect arguments
	// fmt.Println("Number of incorrect arguments: " + strconv.Itoa(flag.NArg()))
	//

	//
	// Only used in client mode, to control whether to display per connection
	// statistics or not.
	//
	gNoConnectionStats = *ncs

	//
	// Only used in client mode to ignore HTTPS cert errors.
	//
	gIgnoreCert = *ic

	xMode := false
	switch *modeStr {
	case "":
	case "x":
		xMode = true
	default:
		cmd.PrintUsageError("Invalid value for execution mode (-m).")
	}
	mode := ethrModeInv

	if *isServer {
		if *clientDest != "" {
			cmd.PrintUsageError("Invalid arguments, \"-c\" cannot be used with \"-s\".")
		}
		if xMode {
			mode = ethrModeExtServer
		} else {
			mode = ethrModeServer
		}
	} else if *clientDest != "" {
		if xMode {
			mode = ethrModeExtClient
		} else {
			mode = ethrModeClient
		}
	} else {
		cmd.PrintUsageError("Invalid arguments, use either \"-s\" or \"-c\".")
	}

	if *reverse && mode != ethrModeClient {
		cmd.PrintUsageError("Invalid arguments, \"-r\" can only be used in client mode.")
	}

	if *use4 && !*use6 {
		ipVer = ethrIPv4
	} else if *use6 && !*use4 {
		ipVer = ethrIPv6
	}

	//Default latency test to 1KB if length is not specified
	switch *bufLenStr {
	case "":
		*bufLenStr = getDefaultBufferLenStr(*testTypePtr)
	}

	bufLen := unitToNumber(*bufLenStr)
	if bufLen == 0 {
		cmd.PrintUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
	}

	if *rttCount <= 0 {
		cmd.PrintUsageError(fmt.Sprintf("Invalid RTT count for latency test: %d", *rttCount))
	}

	var testType EthrTestType
	switch *testTypePtr {
	case "":
		switch mode {
		case ethrModeServer:
			testType = All
		case ethrModeExtServer:
			testType = All
		case ethrModeClient:
			testType = Bandwidth
		case ethrModeExtClient:
			testType = ConnLatency
		}
	case "b":
		testType = Bandwidth
	case "c":
		testType = Cps
	case "p":
		testType = Pps
	case "l":
		testType = Latency
	case "cl":
		testType = ConnLatency
	default:
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
			"Valid parameters and values are:\n", *testTypePtr))
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
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", *protocol))
	}

	if *thCount <= 0 {
		*thCount = runtime.NumCPU()
	}

	//
	// For Pkt/s, we always override the buffer size to be just 1 byte.
	// TODO: Evaluate in future, if we need to support > 1 byte packets for
	//       Pkt/s testing.
	//
	if testType == Pps {
		bufLen = 1
	}

	testParam := EthrTestParam{EthrTestID{EthrProtocol(proto), testType},
		uint32(*thCount),
		uint32(bufLen),
		uint32(*rttCount),
		*reverse}
	validateTestParam(mode, testParam)

	generatePortNumbers(*portStr)

	logFileName := *outputFile
	if !*noOutput {
		if logFileName == defaultLogFileName {
			switch mode {
			case ethrModeServer:
				logFileName = "ethrs.log"
			case ethrModeExtServer:
				logFileName = "ethrxs.log"
			case ethrModeClient:
				logFileName = "ethrc.log"
			case ethrModeExtClient:
				logFileName = "ethrxc.log"
			}
		}
		logInit(logFileName, *debug)
	}

	clientParam := ethrClientParam{*duration, *gap}
	serverParam := ethrServerParam{*showUI}

	switch mode {
	case ethrModeServer:
		runServer(testParam, serverParam)
	case ethrModeExtServer:
		runXServer(testParam, serverParam)
	case ethrModeClient:
		runClient(testParam, clientParam, *clientDest)
	case ethrModeExtClient:
		runXClient(testParam, clientParam, *clientDest)
	}
}

func getDefaultBufferLenStr(testTypePtr string) string {
	if testTypePtr == "l" {
		return latencyDefaultBufferLenStr
	}
	return defaultBufferLenStr
}

func emitUnsupportedTest(testParam EthrTestParam) {
	cmd.PrintUsageError(fmt.Sprintf("\"%s\" test for \"%s\" is not supported.\n",
		testToString(testParam.TestID.Type), protoToString(testParam.TestID.Protocol)))
}

func printReverseModeError() {
	cmd.PrintUsageError("Reverse mode (-r) is only supported for TCP Bandwidth tests.")
}

func validateTestParam(mode ethrMode, testParam EthrTestParam) {
	testType := testParam.TestID.Type
	protocol := testParam.TestID.Protocol
	if mode == ethrModeServer {
		if testType != All || protocol != TCP {
			emitUnsupportedTest(testParam)
		}
	} else if mode == ethrModeClient {
		switch protocol {
		case TCP:
			if testType != Bandwidth && testType != Cps && testType != Latency {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse && testType != Bandwidth {
				printReverseModeError()
			}
		case UDP:
			if testType != Bandwidth && testType != Pps {
				emitUnsupportedTest(testParam)
			}
			if testType == Bandwidth {
				if testParam.BufferSize > (64 * 1024) {
					cmd.PrintUsageError("Maximum supported buffer size for UDP is 64K\n")
				}
			}
			if testParam.Reverse {
				printReverseModeError()
			}
		case HTTP:
			if testType != Bandwidth && testType != Latency {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse {
				printReverseModeError()
			}
		case HTTPS:
			if testType != Bandwidth {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse {
				printReverseModeError()
			}
		default:
			emitUnsupportedTest(testParam)
		}
	} else if mode == ethrModeExtClient {
		if (protocol != TCP) || (testType != ConnLatency && testType != Bandwidth) {
			emitUnsupportedTest(testParam)
		}
	}
}
