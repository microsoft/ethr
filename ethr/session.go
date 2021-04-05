package ethr

import "net"

type Session interface {
	CreateOrGetTest(string, Protocol, TestType) (*Test, bool)
	SafeDeleteTest(*Test) bool
	Send(net.Conn, *Msg) error
	Receive(net.Conn) *Msg
}
