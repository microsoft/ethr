package tools

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"weavelab.xyz/ethr/ethr"
)

type Tools struct {
	IPVersion ethr.IPVersion

	IsExternal     bool
	RemoteIP       net.IP
	RemotePort     uint16
	RemoteHostname string
	RemoteRaw      string

	LocalPort uint16
	LocalIP   net.IP
}

func NewTools(isExternal bool, remote string, localPort uint16, localIP net.IP) (*Tools, error) {
	rHostname, rIP, rPort, err := getServerIPandPort(isExternal, remote)
	if err != nil {
		return nil, fmt.Errorf("error parsing server host and port (%s): %w", remote, err)
	}
	var ipVersion ethr.IPVersion
	ip := net.ParseIP(rIP)
	if ip != nil {
		if ip.To4() != nil {
			ipVersion = ethr.IPv4
		} else {
			ipVersion = ethr.IPv6
		}
	} else {
		return nil, fmt.Errorf("failed to parse server IP from (%s)", rIP)
	}

	return &Tools{
		IPVersion:      ipVersion,
		IsExternal:     isExternal,
		RemoteIP:       ip,
		RemotePort:     rPort,
		RemoteHostname: rHostname,
		RemoteRaw:      remote,
		LocalPort:      localPort,
		LocalIP:        localIP,
	}, nil
}

func getServerIPandPort(isExternal bool, remote string) (string, string, uint16, error) {
	hostName := ""
	hostIP := ""
	port := ""
	u, err := url.Parse(remote)
	if err == nil && u.Hostname() != "" {
		hostName = u.Hostname()
		if u.Port() != "" {
			port = u.Port()
		} else {
			// Only implicitly derive port in External client mode.
			if isExternal {
				if u.Scheme == "http" {
					port = "80"
				} else if u.Scheme == "https" {
					port = "443"
				}
			}
		}
	} else {
		hostName, port, err = net.SplitHostPort(remote)
		if err != nil {
			hostName = remote
		}
	}
	outPort, err := strconv.Atoi(port)
	if err != nil {
		return hostName, hostIP, 0, fmt.Errorf("failed to parse port: %w", err)
	}

	_, hostIP, err = lookupIP(hostName)
	if err != nil {
		return hostName, hostIP, 0, fmt.Errorf("failed to resolve ip: %w", err)
	}
	return hostName, hostIP, uint16(outPort), nil
}

func lookupIP(remote string) (net.IPAddr, string, error) {
	var ipAddr net.IPAddr
	var ipStr string

	ip := net.ParseIP(remote)
	if ip != nil {
		ipAddr.IP = ip
		ipStr = remote
		return ipAddr, ipStr, nil
	}

	ips, err := net.LookupIP(remote)
	if err != nil {
		return ipAddr, ipStr, fmt.Errorf("failed to lookup IP address for the server: %v. Error: %w", remote, err)
	}
	for _, ip := range ips {
		if ethr.CurrentIPVersion == ethr.IPAny || (ethr.CurrentIPVersion == ethr.IPv4 && ip.To4() != nil) || (ethr.CurrentIPVersion == ethr.IPv6 && ip.To16() != nil) {
			ipAddr.IP = ip
			ipStr = ip.String()
			return ipAddr, ipStr, nil
		}
	}
	return ipAddr, ipStr, fmt.Errorf("unable to resolve the given server: %v to an IP address: %w", remote, os.ErrNotExist)
}
