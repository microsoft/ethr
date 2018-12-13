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
	clientServerIP := flag.String("c", "",
		"Run as client and connect to server specified by String")
	testTypePtr := flag.String("t", "b",
		"Test to run (\"b\", \"c\", \"p\" or \"l\")\n"+
			"b: Bandwidth\n"+
			"c: Connections/s or Requests/s\n"+
			"p: Packets/s\n"+
			"l: Latency, Loss & Jitter")
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
		"Number of round trip iterations for calculating latency.")
	ethrUnused(noOutput)

	flag.Parse()

	//
	// TODO: Handle the case if there are incorrect arguments
	// fmt.Println("Number of incorrect arguments: " + strconv.Itoa(flag.NArg()))
	//

	if (*isServer && *clientServerIP != "") ||
		(!*isServer && *clientServerIP == "") {
		fmt.Println("Please specify either server mode (-s) or client mode (-c).")
		flag.PrintDefaults()
		os.Exit(1)
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
	case "b":
		testType = Bandwidth
	case "c":
		testType = Cps
	case "p":
		testType = Pps
	case "l":
		testType = Latency
	default:
		fmt.Printf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
			"Valid parameters and values are:\n", *testTypePtr)
		flag.PrintDefaults()
		os.Exit(1)
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
		fmt.Printf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", *protocol)
		flag.PrintDefaults()
		os.Exit(1)
	}

	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		fmt.Printf("Invalid value \"%s\" specified for parameter \"-d\".\n",
			*durationStr)
		flag.PrintDefaults()
		os.Exit(1)
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
	if !validateTestParam(testParam) {
		os.Exit(1)
	}

	logFileName := *outputFile
	if *isServer {
		if !*noOutput {
			if logFileName == defaultLogFileName {
				logFileName = "ethrs.log"
			}
			logInit(logFileName, *debug)
		}
		runServer(testParam, *showUI)
	} else {
		if !*noOutput {
			if logFileName == defaultLogFileName {
				logFileName = "ethrc.log"
			}
			logInit(logFileName, *debug)
		}
		runClient(testParam, *clientServerIP, duration)
	}
}

func emitUnsupportedTest(testParam EthrTestParam) {
	fmt.Printf("Error: \"%s\" test for \"%s\" is not supported.\n",
		testToString(testParam.TestID.Type), protoToString(testParam.TestID.Protocol))
}

func validateTestParam(testParam EthrTestParam) bool {
	testType := testParam.TestID.Type
	protocol := testParam.TestID.Protocol
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
	default:
		emitUnsupportedTest(testParam)
		return false
	}
	return true
}
