package log

import (
	"net"

	"weavelab.xyz/ethr/ethr"
)

type AggregateLogger struct {
	loggers []ethr.Logger
}

func NewAggregateLogger(loggers ...ethr.Logger) *AggregateLogger {
	return &AggregateLogger{loggers: loggers}
}

func (l *AggregateLogger) Error(format string, args ...interface{}) {
	for _, logger := range l.loggers {
		logger.Error(format, args...)
	}
}

func (l *AggregateLogger) Info(format string, args ...interface{}) {
	for _, logger := range l.loggers {
		logger.Info(format, args...)
	}
}

func (l *AggregateLogger) Debug(format string, args ...interface{}) {
	for _, logger := range l.loggers {
		logger.Debug(format, args...)
	}
}

func (l *AggregateLogger) TestResult(tt ethr.TestType, success bool, protocol ethr.Protocol, rIP net.IP, rPort uint16, result interface{}) {
	for _, logger := range l.loggers {
		logger.TestResult(tt, success, protocol, rIP, rPort, result)
	}
}
