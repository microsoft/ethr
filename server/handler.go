package server

import (
	"context"
	"net"

	"weavelab.xyz/ethr/session"
)

type Handler interface {
	HandleConn(context.Context, *session.Test, net.Conn)
}
