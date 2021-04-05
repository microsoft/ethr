package server

import "weavelab.xyz/ethr/ethr"

type Config struct {
	IPVersion ethr.IPVersion
	LocalIP string
	LocalPort string
}