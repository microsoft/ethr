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

func main() {
	isServer := flag.Bool("s", false, "Run as server")
	clientDest := flag.String("c", "",
		"Run as client and connect to server specified by String\n"+
			"Ethr mode (default): specifies host name or IP address for Ethr server\n"+
			"External mode: specifies IP:port or name:port or URI:port for external server")
	testTypePtr := flag.String("t", "",
		"Test to run (\"b\", \"c\", \"p\", \"l\" or \"cl\")\n"+
			"b: Bandwidth\n"+
			"c: Connections/s or Requests/s\n"+
			"p: Packets/s\n"+
			"l: Latency, Loss & Jitter\n"+
			"cl: Connection setup latency\n"+
			"Default: b - Regular mode, cl - External mode")
	thCount := flag.Int("n", 1,
		"Number of Threads\n"+
			"0: Equal to number of CPUs")
	bufLenStr := flag.String("l", "16KB",
		"Length of buffer to use (format: <num>[KB | MB | GB])\n"+
			"Only valid for Bandwidth tests. Max 1GB.")
	protocol := flag.String("p", "tcp",
		"Protocol (\"tcp\", \"udp\", \"http\", \"https\", or \"icmp\")")
	outputFile := flag.String("o", defaultLogFileName,
		"Name of the file for logging output.\n")
	debug := flag.Bool("debug", false, "Log debug output. Only valid if \"-o\" is specified.")
	noOutput := flag.Bool("no", false, "Disable logging output to file.")
	durationStr := flag.String("d", "10s",
		"Duration for the test (format: <num>[s | m | h] \n"+
			"0: Run forever")
	showUI := flag.Bool("ui", false, "Show output in text UI. Valid for server only.")
	rttCount := flag.Int("i", 1000,
		"Number of round trip iterations for latency test.")
	portStr := flag.String("ports", "",
		"Ports to use for server and client\n"+
			"Format: \"key1=value1, key2=value2\"\n"+
			"Example: \"control=8888, tcp=9999, http=8099\"\n"+
			"For protocols, only base port is specified, so tcp=9999 means:\n"+
			"9999 - Bandwidth, 9998 - CPS, 9997 - PPS, 9996 - Latency tests\n"+
			"Default: control=8888, tcp=9999, udp=9999, http=9899, https=9799")
	modeStr := flag.String("m", "",
		"Execution mode for Ethr (\"\" or \"x\")\n"+
			"Default: Ethr mode\n"+
			"x: External mode\n"+
			"In external mode, ethr client connects to non-ethr server")
	use4 := flag.Bool("4", false, "Use IPv4 only")
	use6 := flag.Bool("6", false, "Use IPv6 only")

	flag.Parse()

	//
	// TODO: Handle the case if there are incorrect arguments
	// fmt.Println("Number of incorrect arguments: " + strconv.Itoa(flag.NArg()))
	//

	xMode := false
	switch *modeStr {
	case "":
	case "x":
		xMode = true
	default:
		printUsageError("Invalid value for execution mode (-m).")
	}
	mode := ethrModeInv

	if *isServer {
		if *clientDest != "" {
			printUsageError("Invalid arguments, \"-c\" cannot be used with \"-s\".")
		}
		if xMode {
			printUsageError("External mode is not supported for server (yet).")
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
		printUsageError("Invalid arguments, use either \"-s\" or \"-c\".")
	}

	if *use4 && !*use6 {
		ipVer = ethrIPv4
	} else if *use6 && !*use4 {
		ipVer = ethrIPv6
	}

	bufLen := unitToNumber(*bufLenStr)
	if bufLen == 0 {
		fmt.Println("Invalid length specified: " + *bufLenStr)
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *rttCount <= 0 {
		fmt.Println("Invalid RTT count for latency test:", *rttCount)
		flag.PrintDefaults()
		os.Exit(1)
	}

	var testType EthrTestType
	switch *testTypePtr {
	case "":
		switch mode {
		case ethrModeServer:
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
	default:
		printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
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
		printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", *protocol))
	}

	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		printUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-d\".\n", *durationStr))
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
		uint32(*rttCount)}
	if !validateTestParam(mode, testParam) {
		os.Exit(1)
	}

	generatePortNumbers(*portStr)

	logFileName := *outputFile
	if !*noOutput {
		if logFileName == defaultLogFileName {
			switch mode {
			case ethrModeServer:
				logFileName = "ethrs.log"
			case ethrModeClient:
				logFileName = "ethrc.log"
			case ethrModeExtClient:
				logFileName = "ethrxc.log"
			}
		}
		logInit(logFileName, *debug)
	}

	clientParam := ethrClientParam{duration}
	serverParam := ethrServerParam{*showUI}

	switch mode {
	case ethrModeServer:
		runServer(testParam, serverParam)
	case ethrModeClient:
		runClient(testParam, clientParam, *clientDest)
	case ethrModeExtClient:
		runXClient(testParam, clientParam, *clientDest)
	}
}

func emitUnsupportedTest(testParam EthrTestParam) {
	fmt.Printf("Error: \"%s\" test for \"%s\" is not supported.\n",
		testToString(testParam.TestID.Type), protoToString(testParam.TestID.Protocol))
}

func validateTestParam(mode ethrMode, testParam EthrTestParam) bool {
	testType := testParam.TestID.Type
	protocol := testParam.TestID.Protocol
	if mode == ethrModeServer {
		if testType != All || protocol != TCP {
			emitUnsupportedTest(testParam)
			return false
		}
	} else if mode == ethrModeClient {
		switch protocol {
		case TCP:
			if testType != Bandwidth && testType != Cps && testType != Latency {
				emitUnsupportedTest(testParam)
				return false
			}
		case UDP:
			if testType != Bandwidth && testType != Pps {
				emitUnsupportedTest(testParam)
				return false
			}
			if testType == Bandwidth {
				if testParam.BufferSize > (64 * 1024) {
					fmt.Printf("Error: Maximum supported buffer size for UDP is 64K\n")
					return false
				}
			}
		case HTTP:
			if testType != Bandwidth {
				emitUnsupportedTest(testParam)
				return false
			}
		case HTTPS:
			if testType != Bandwidth {
				emitUnsupportedTest(testParam)
				return false
			}
		default:
			emitUnsupportedTest(testParam)
			return false
		}
	} else if mode == ethrModeExtClient {
		if testType != ConnLatency || protocol != TCP {
			emitUnsupportedTest(testParam)
			return false
		}
	}
	return true
}

func printUsageError(s string) {
	fmt.Printf("Error: %s\n", s)
	flag.PrintDefaults()
	os.Exit(1)
}
