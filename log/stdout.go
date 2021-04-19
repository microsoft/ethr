package log

import (
	"context"
	"fmt"
	"net"

	"weavelab.xyz/ethr/ethr"
)

type STDOutLogger struct {
	ll     LogLevel
	active bool
	toLog  chan string
}

func NewSTDOutLogger(ll LogLevel) *STDOutLogger {
	return &STDOutLogger{
		ll:    ll,
		toLog: make(chan string, 64), // TODO determine adequate buffer
	}
}

func (l *STDOutLogger) Init(ctx context.Context) {
	go l.writeLogs(ctx)
}

func (l *STDOutLogger) writeLogs(ctx context.Context) {
	l.active = true

	for line := range l.toLog {
		select {
		case <-ctx.Done():
			l.active = false
			return
		default:
			// do nothing
		}

		if l.active {
			fmt.Println(line)
		}
	}
	l.active = false
}

func (l *STDOutLogger) queueMessage(msg Message) {
	if l.active {
		l.toLog <- fmt.Sprintf("[%s] %s - %s", msg.Level.String(), msg.Timestamp, msg.Message)
	}
}

func (l *STDOutLogger) Error(format string, args ...interface{}) {
	l.queueMessage(NewMessage(LevelError, fmt.Sprintf(format, args...)))
}

func (l *STDOutLogger) Info(format string, args ...interface{}) {
	l.queueMessage(NewMessage(LevelInfo, fmt.Sprintf(format, args...)))

}

func (l *STDOutLogger) Debug(format string, args ...interface{}) {
	if l.ll == LevelDebug {
		l.queueMessage(NewMessage(LevelDebug, fmt.Sprintf(format, args...)))
	}
}

var NoDetails StaticStringer

type StaticStringer string

func (s StaticStringer) String() string {
	return "UNKNOWN DETAILS"
}

func (l *STDOutLogger) TestResult(tt ethr.TestType, success bool, protocol ethr.Protocol, rIP net.IP, rPort uint16, body interface{}) {
	if l.active {
		status := "FAILURE"
		if success {
			status = "SUCCESS"
		}
		result, ok := body.(fmt.Stringer)
		if !ok {
			result = NoDetails
		}
		l.toLog <- fmt.Sprintf("[RESULT] %s: %s - %s:%d (%s):: %s", tt, status, rIP, rPort, protocol, result)
	}
}
