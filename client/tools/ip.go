package tools

import "net"

func (t Tools) LookupIP(remote string) (net.IPAddr, string, error) {
	return lookupIP(remote)
}
