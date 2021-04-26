package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"time"

	"weavelab.xyz/ethr/ui"

	"weavelab.xyz/ethr/ethr"
)

var Version = "UNKNOWN"

var (
	NoOutput   bool
	OutputFile string
	Debug      bool
	UseIPv4    bool
	UseIPv6    bool
	IPVersion  ethr.IPVersion
	Port       uint16
	LocalIP    net.IP
	IsServer   bool

	// Server Only
	ShowUI bool

	// Client Only
	ClientDest         string
	RemoteIP           net.IP
	BufferSize         uint64
	BandwidthRate      uint64
	LocalPort          uint16
	Duration           time.Duration
	Gap                time.Duration
	Iterations         int
	NoConnectionStats  bool
	Protocol           ethr.Protocol
	Reverse            bool
	TestType           ethr.TestType
	TOS                int
	Title              string
	ThreadCount        int
	WarmupCount        int
	IsExternal         bool
	ExternalClientDest string

	// Tuning
	LogBufferSize int
)

var hasPortRegex = regexp.MustCompile(".+(:\\d+)")

func Init() error {
	flag.Usage = func() { Usage() }
	flag.BoolVar(&NoOutput, "no", false, "")
	flag.StringVar(&OutputFile, "o", "", "")
	flag.BoolVar(&Debug, "debug", false, "")
	flag.BoolVar(&UseIPv4, "4", false, "")
	flag.BoolVar(&UseIPv6, "6", false, "")
	port := flag.Int("port", 8888, "")
	rawIP := flag.String("ip", "localhost", "")
	flag.BoolVar(&IsServer, "s", false, "")

	flag.BoolVar(&ShowUI, "ui", false, "")

	flag.StringVar(&ClientDest, "c", "", "")
	bufferLen := flag.String("l", "", "")
	bw := flag.String("b", "", "")
	lport := flag.Int("cport", 9999, "")
	flag.DurationVar(&Duration, "d", 10*time.Second, "")
	flag.DurationVar(&Gap, "g", time.Second, "")
	flag.IntVar(&Iterations, "i", 1000, "")
	flag.BoolVar(&NoConnectionStats, "ncs", false, "")
	rawProtocol := flag.String("p", "tcp", "")
	flag.BoolVar(&Reverse, "r", false, "")
	rawTestType := flag.String("t", "b", "")
	flag.IntVar(&TOS, "tos", 0, "")
	flag.StringVar(&Title, "T", "", "")
	flag.IntVar(&ThreadCount, "n", 0, "")
	flag.IntVar(&WarmupCount, "w", 1, "")
	flag.StringVar(&ExternalClientDest, "x", "", "")

	flag.IntVar(&LogBufferSize, "logbuffer", 64, "maximum number of lines buffered in logger")

	flag.Parse()

	LocalPort = uint16(*lport)
	Port = uint16(*port)

	// MUST set ip version before resolving IPs to resolve properly
	if (!UseIPv4 && !UseIPv6) || (UseIPv4 && UseIPv6) {
		IPVersion = ethr.IPAny
	} else if UseIPv6 {
		IPVersion = ethr.IPv6
	} else {
		IPVersion = ethr.IPv4
	}

	IsExternal = ExternalClientDest != ""

	var err error
	LocalIP, err = lookupIP(*rawIP)
	if err != nil {
		return fmt.Errorf("failed to determine local IP: %w", err)
	}

	if IsExternal {
		RemoteIP, err = lookupIP(ExternalClientDest)
		if err != nil {
			return fmt.Errorf("failed to determine remote IP: %w", err)
		}
	} else {
		RemoteIP, err = lookupIP(ClientDest)
		if err != nil {
			return fmt.Errorf("failed to determine remote IP: %w", err)
		}
	}

	if OutputFile == "" {
		if IsServer {
			OutputFile = "ethrs.log"
		} else {
			OutputFile = "ethrc.log"
		}
	}

	Protocol = ethr.ParseProtocol(*rawProtocol)
	if Protocol == ethr.ProtocolUnknown {
		return fmt.Errorf("invalid protocol: %s", *rawProtocol)
	}

	TestType = ethr.ParseTestType(*rawTestType)
	if IsServer {
		TestType = ethr.TestTypeServer
	} else if TestType == ethr.TestTypeUnknown {
		return errors.New("invalid test type")
	}

	if !IsServer {
		if *bufferLen == "" {
			if TestType == ethr.TestTypeLatency || TestType == ethr.TestTypePacketsPerSecond {
				BufferSize = ui.UnitToNumber("1B")
			} else {
				BufferSize = ui.UnitToNumber("16KB")
			}
		} else {
			BufferSize = ui.UnitToNumber(*bufferLen)
		}
		if BufferSize == 0 {
			return errors.New("invalid buffer size")
		}

		if *bw != "" {
			BandwidthRate = ui.UnitToNumber(*bw) / 8
		}

		if ThreadCount == 0 {
			ThreadCount = runtime.NumCPU()
		}
	}

	Debug = true

	if IsServer {
		return validateServerArgs()
	}
	return validateClientArgs()
}

func validateServerArgs() (err error) {
	invalidFlags := make([]string, 0)
	if ClientDest != "" {
		invalidFlags = append(invalidFlags, "-c")
	}
	if ExternalClientDest != "" {
		invalidFlags = append(invalidFlags, "-x")
	}
	if BufferSize != 0 {
		invalidFlags = append(invalidFlags, "-l")
	}
	if BandwidthRate != 0 {
		invalidFlags = append(invalidFlags, "-b")
	}
	if LocalPort != 0 {
		invalidFlags = append(invalidFlags, "-cport")
	}
	if Duration != 10*time.Second {
		invalidFlags = append(invalidFlags, "-d")
	}
	if Gap != time.Second {
		invalidFlags = append(invalidFlags, "-g")
	}
	if Iterations != 1000 {
		invalidFlags = append(invalidFlags, "-i")
	}
	if NoConnectionStats {
		invalidFlags = append(invalidFlags, "-ncs")
	}
	if Protocol != ethr.TCP {
		invalidFlags = append(invalidFlags, "-p")
	}
	if Reverse {
		invalidFlags = append(invalidFlags, "-r")
	}
	if TestType != ethr.TestTypeServer {
		invalidFlags = append(invalidFlags, "-t")
	}
	if TOS != 0 {
		invalidFlags = append(invalidFlags, "-tos")
	}
	if ThreadCount != 0 {
		invalidFlags = append(invalidFlags, "-n")
	}
	if WarmupCount != 1 {
		invalidFlags = append(invalidFlags, "-wc")
	}
	if Title != "" {
		invalidFlags = append(invalidFlags, "-T")
	}

	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid command, %s can only be used in client (\"-c\") mode", invalidFlags)
	}

	return nil
}

func validateClientArgs() error {
	if ShowUI {
		return fmt.Errorf("invalid argument, -ui can only be used in server (\"-s\") mode")
	}
	if ClientDest != "" && ExternalClientDest != "" {
		return fmt.Errorf("invalid argument, both \"-c\" and \"-x\" cannot be specified at the same time")
	}
	if IsExternal && Protocol == ethr.TCP && Port == 0 {
		return fmt.Errorf("in external mode, port cannot be empty for TCP tests")
	}

	if ThreadCount < 1 {
		return fmt.Errorf("must use at least 1 thread")
	}

	// Validate protocol, test type, and params configuration for tests
	if IsExternal {
		if Protocol == ethr.TCP {
			switch TestType {
			case ethr.TestTypePing, ethr.TestTypeConnectionsPerSecond, ethr.TestTypeTraceRoute, ethr.TestTypeMyTraceRoute:
			default:
				return unsupportedTest()
			}
		} else if Protocol == ethr.ICMP {
			switch TestType {
			case ethr.TestTypePing, ethr.TestTypeTraceRoute, ethr.TestTypeMyTraceRoute:
			default:
				return unsupportedTest()
			}
		} else {
			return unsupportedTest()
		}
	} else {
		if Reverse && TestType != ethr.TestTypeBandwidth {
			return fmt.Errorf("reverse mode (-r) is only supported for TCP Bandwidth tests")
		}

		switch Protocol {
		case ethr.TCP:
			switch TestType {
			case ethr.TestTypeBandwidth, ethr.TestTypeConnectionsPerSecond, ethr.TestTypeLatency, ethr.TestTypePing, ethr.TestTypeTraceRoute, ethr.TestTypeMyTraceRoute:
				if BufferSize > 2*ui.GIGA {
					return fmt.Errorf("maximum tcp buffer size is 2GB")
				}
			default:
				return unsupportedTest()
			}
		case ethr.UDP:
			switch TestType {
			case ethr.TestTypeBandwidth, ethr.TestTypePacketsPerSecond:
				if BufferSize > 64*ui.KILO {
					return fmt.Errorf("maximum udp buffer is 64KB")
				}
			default:
				return unsupportedTest()
			}
		//case ethr.ICMP:
		//	switch TestType {
		//	case ethr.TestTypePing, ethr.TestTypeTraceRoute, ethr.TestTypeMyTraceRoute:
		//	default:
		//		return unsupportedTest()
		//	}
		default:
			return unsupportedTest()
		}
	}

	return nil
}

func unsupportedTest() error {
	return fmt.Errorf("unsupported test/protocol: (%s/%s)", TestType, Protocol)
}

func lookupIP(remote string) (addr net.IP, err error) {
	addr = net.ParseIP(remote)
	if addr != nil {
		return
	}

	ips, err := net.LookupIP(remote)
	if err != nil {
		err = fmt.Errorf("failed to lookup IP for server %s: %w", remote, err)
		return
	}
	for _, ip := range ips {
		if IPVersion == ethr.IPAny || (IPVersion == ethr.IPv4 && ip.To4() != nil) || (IPVersion == ethr.IPv6 && ip.To16() != nil) {
			addr = ip
			return
		}
	}
	err = fmt.Errorf("unable to resolve ip of server %s: %w", remote, os.ErrNotExist)
	return
}

func GetAddrString(ip net.IP, port uint16) string {
	if ip == nil {
		return fmt.Sprintf(":%d", port) // listen on localhost for IPv4 AND IPv6 :/
	}
	if port == 0 {
		return ip.String()
	}
	if ip.To16() != nil {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}
