package log

import (
	"fmt"
	"net"
	"time"

	"weavelab.xyz/ethr/ethr"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
)

func (l LogLevel) MarshalJSON() ([]byte, error) {
	switch l {
	case LevelError:
		return []byte("ERROR"), nil
	case LevelInfo:
		return []byte("INFO"), nil
	case LevelDebug:
		return []byte("DEBUG"), nil
	default:
		return []byte("UNKNOWN"), nil
	}
}

func (l LogLevel) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

type Message struct {
	Timestamp string
	Level     LogLevel
	Message   string
}

func NewMessage(ll LogLevel, msg string) Message {
	return Message{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     ll,
		Message:   msg,
	}
}

type TestResultLog struct {
	Timestamp string
	Type      ethr.TestType
	Protocol  ethr.Protocol
	Remote    string
	Success   bool
	Details   interface{}
}

func NewTestResultLog(tt ethr.TestType, success bool, protocol ethr.Protocol, rIP net.IP, rPort uint16, details interface{}) TestResultLog {
	return TestResultLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Type:      tt,
		Protocol:  protocol,
		Remote:    fmt.Sprintf("%s:%d", rIP.String(), rPort),
		Success:   success,
		Details:   details,
	}
}
