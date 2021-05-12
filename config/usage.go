package config

import "fmt"

// Usage prints the command-line usage text
func Usage() {
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
