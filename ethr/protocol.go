package ethr

import "strings"

type Protocol uint32

const (
	TCP Protocol = iota
	UDP
	ICMP
	HTTP
	HTTPS
	ProtocolUnknown
)

const (
	ICMPv4 = 1  // ICMP for IPv4
	ICMPv6 = 58 // ICMP for IPv6
)

func (p Protocol) String() string {
	switch p {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	case ICMP:
		return "ICMP"
	case HTTP:
		return "HTTP"
	case HTTPS:
		return "HTTPS"
	}
	return ""
}

func ParseProtocol(s string) Protocol {
	switch strings.ToUpper(s) {
	case "TCP":
		return TCP
	case "UDP":
		return UDP
	case "ICMP":
		return ICMP
	case "HTTP":
		return HTTP
	case "HTTPS":
		return HTTPS
	}
	return ProtocolUnknown
}

func TCPVersion(v IPVersion) string {
	if v == IPv4 {
		return "tcp4"
	} else if v == IPv6 {
		return "tcp6"
	}
	return "tcp"
}

func ICMPVersion(v IPVersion) string {
	if v == IPv6 {
		return "ip6:ipv6-icmp"
	}
	return "ip4:icmp"
}

func UDPVersion(v IPVersion) string {
	if v == IPv4 {
		return "udp4"
	} else if v == IPv6 {
		return "udp6"
	}
	return "udp"
}

func ICMPProtocolNumber(v IPVersion) int {
	if v == IPv6 {
		return ICMPv6
	}
	return ICMPv4
}
