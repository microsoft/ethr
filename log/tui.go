package log

import (
	"context"
	"fmt"
	"net"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/ui/server"
)

type TuiLogger struct {
	ui     server.ServerUI
	ll     LogLevel
	active bool
}

func NewTuiLogger(ll LogLevel, ui server.ServerUI) *TuiLogger {
	return &TuiLogger{
		ui: ui,
		ll: ll,
	}
}

func (l *TuiLogger) Init(ctx context.Context) {
	l.active = true
	go func() {
		<-ctx.Done()
		l.active = false
	}()
}

func (l *TuiLogger) Error(format string, args ...interface{}) {
	if l.active {
		l.ui.AddErrorMsg(fmt.Sprintf(format, args...))
	}
}

func (l *TuiLogger) Info(format string, args ...interface{}) {
	if l.active {
		l.ui.AddInfoMsg(fmt.Sprintf(format, args...))
	}
}

func (l *TuiLogger) Debug(format string, args ...interface{}) {
	if l.ll == LevelDebug && l.active {
		l.ui.AddInfoMsg(fmt.Sprintf(format, args...))
	}
}

func (l *TuiLogger) TestResult(tt ethr.TestType, success bool, protocol ethr.Protocol, rIP net.IP, rPort uint16, result interface{}) {
	// do nothing
}
