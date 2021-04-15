package server

import (
	"net"

	"weavelab.xyz/ethr/ethr"
)

type Config struct {
	IPVersion ethr.IPVersion
	LocalIP   net.IP
	LocalPort int
}
