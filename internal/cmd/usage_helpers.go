package cmd

import (
	"fmt"
	"os"
)

func PrintFlagUsage(flag, info string, helptext ...string) {
	fmt.Printf("\t-%s %s\n", flag, info)
	for _, help := range helptext {
		fmt.Printf("\t\t%s\n", help)
	}
}

func PrintServerUsage() {
	PrintFlagUsage("s", "", "Run in server mode.")
}

func PrintClientUsage() {
	PrintFlagUsage("c", "<server>", "Run in client mode and connect to <server>.",
		"Server is specified using name, FQDN or IP address.")
}

func PrintExtClientUsage() {
	PrintFlagUsage("c", "<destination>", "Run in external client mode and connect to <destination>.",
		"<destination> is specified using host:port format.",
		"Example: www.microsoft.com:443 or 10.1.0.4:22 etc.")
}

func PrintPortUsage() {
	PrintFlagUsage("ports", "<k=v,...>", "Use custom port numbers instead of default ones.",
		"A comma separated list of key=value pair is used.",
		"Key specifies the protocol, and value specifies base port.",
		"Ports used for various tests are calculated from base port.",
		"Example: For TCP, Bw: 9999, CPS: 9998, PPS: 9997, Latency: 9996",
		"Control is used for control channel communication for ethr.",
		"Note: Same configuration must be used on both client & server.",
		"Default: 'control=8888,tcp=9999,udp=9999,http=9899,https=9799'")
}

func PrintExtPortUsage() {
	PrintFlagUsage("ports", "<k=v,...>", "Use custom port numbers instead of default ones.",
		"A comma separated list of key=value pair is used.",
		"Key specifies the protocol, and value specifies the port.",
		"Default: 'tcp=9999,http=9899,https=9799'")
}

func PrintTestType() {
	PrintFlagUsage("t", "<test>", "Test to run (\"b\", \"c\", \"p\", or \"l\")",
		"b: Bandwidth",
		"c: Connections/s or Requests/s",
		"p: Packets/s",
		"l: Latency, Loss & Jitter",
		"Default: b - Bandwidth measurement.")
}

func PrintExtTestType() {
	PrintFlagUsage("t", "<test>", "Test to run (\"b\", \"c\", or \"cl\")",
		"b: Bandwidth",
		"c: Connections/s or Requests/s",
		"cl: TCP connection setup latency",
		"Default: cl - TCP connection setup latency.")
}

func PrintThreadUsage() {
	PrintFlagUsage("n", "<number>", "Number of Parallel Sessions (and Threads).",
		"0: Equal to number of CPUs",
		"Default: 1")
}

func PrintDurationUsage() {
	PrintFlagUsage("d", "<duration>",
		"Duration for the test (format: <num>[ms | s | m | h]",
		"0: Run forever",
		"Default: 10s")
}

func PrintGapUsage() {
	PrintFlagUsage("g", "<gap>",
		"Time interval between successive measurements (format: <num>[ms | s | m | h]",
		"0: No gap",
		"Default: 1s")
}

func PrintBufLenUsage() {
	PrintFlagUsage("l", "<length>",
		"Length of buffer to use (format: <num>[KB | MB | GB])",
		"Only valid for Bandwidth tests. Max 1GB.",
		"Default: 16KB")
}

func PrintProtocolUsage() {
	PrintFlagUsage("p", "<protocol>",
		"Protocol (\"tcp\", \"udp\", \"http\", \"https\", or \"icmp\")",
		"Default: tcp")
}

func PrintExtProtocolUsage() {
	PrintFlagUsage("p", "<protocol>",
		"Protocol (\"tcp\", \"http\", \"https\", or \"icmp\")",
		"Default: tcp")
}

func PrintIterationUsage() {
	PrintFlagUsage("i", "<iterations>",
		"Number of round trip iterations for each latency measurement.",
		"Default: 1000")
}

func PrintModeUsage() {
	PrintFlagUsage("m", "<mode>",
		"'-m x' MUST be specified for external mode.")
}

func PrintNoConnStatUsage() {
	PrintFlagUsage("ncs", "",
		"No per Connection Stats would be printed if this flag is specified.",
		"This is useful to suppress verbose logging when large number of",
		"connections are used as specified by -n option for Bandwidth tests.")
}

func PrintIgnoreCertUsage() {
	PrintFlagUsage("ic", "",
		"Ignore Certificate is useful for HTTPS tests, for cases where a",
		"middle box like a proxy is not able to supply a valid Ethr cert.")
}

func PrintUsageError(s string) {
	fmt.Printf("Error: %s\n", s)
	fmt.Printf("Please use \"ethr -h\" for ethr command line arguments.\n")
	os.Exit(1)
}
