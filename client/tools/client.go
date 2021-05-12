package tools

import (
	"net"

	"weavelab.xyz/ethr/ethr"
)

type Tools struct {
	IPVersion ethr.IPVersion
	Logger    ethr.Logger

	IsExternal bool
	RemoteIP   net.IP
	RemotePort uint16

	LocalPort uint16
	LocalIP   net.IP
}

func NewTools(isExternal bool, rIP net.IP, rPort uint16, localPort uint16, localIP net.IP, logger ethr.Logger) (*Tools, error) {
	var ipVersion ethr.IPVersion
	if rIP != nil {
		if rIP.To4() != nil {
			ipVersion = ethr.IPv4
		} else {
			ipVersion = ethr.IPv6
		}
	}
	//else {
	//	return nil, fmt.Errorf("failed to parse server IP from (%s)", rIP)
	//}

	return &Tools{
		IPVersion:  ipVersion,
		IsExternal: isExternal,
		RemoteIP:   rIP,
		RemotePort: rPort,
		LocalPort:  localPort,
		LocalIP:    localIP,
		Logger:     logger,
	}, nil
}
