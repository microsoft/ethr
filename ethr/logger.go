package ethr

import "net"

type Logger interface {
	Error(string, ...interface{})
	Info(string, ...interface{})
	Debug(string, ...interface{})
	TestResult(TestType, bool, Protocol, net.IP, uint16, interface{})
}
