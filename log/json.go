package log

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"weavelab.xyz/ethr/ethr"
)

type JSONLogger struct {
	logFile os.File
	ll      LogLevel
	active  bool
	toLog   chan string
}

func NewJSONLogger(filename string, ll LogLevel, bufferSize int) (*JSONLogger, error) {
	if filename == "" {
		return nil, errors.New("filename required")
	}
	logFile, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to open the log file (%s): %w", filename, err)
	}

	log.SetFlags(0)
	log.SetOutput(logFile)
	return &JSONLogger{
		ll:    ll,
		toLog: make(chan string, bufferSize),
	}, nil
}

func (l *JSONLogger) Init(ctx context.Context) {
	go l.writeLogs(ctx)
}

func (l *JSONLogger) writeLogs(ctx context.Context) {
	l.active = true
	defer l.logFile.Close()

	for line := range l.toLog {
		select {
		case <-ctx.Done():
			l.active = false
			return
		default:
			// do nothing
		}

		if l.active {
			log.Println(line)
		}
	}
	l.active = false
}

func (l *JSONLogger) queueMessage(msg Message) {
	if l.active {
		lineJson, _ := json.Marshal(msg)
		l.toLog <- string(lineJson)
	}
}

func (l *JSONLogger) Error(format string, args ...interface{}) {
	l.queueMessage(NewMessage(LevelError, fmt.Sprintf(format, args...)))
}

func (l *JSONLogger) Info(format string, args ...interface{}) {
	l.queueMessage(NewMessage(LevelInfo, fmt.Sprintf(format, args...)))

}

func (l *JSONLogger) Debug(format string, args ...interface{}) {
	if l.ll == LevelDebug {
		l.queueMessage(NewMessage(LevelDebug, fmt.Sprintf(format, args...)))
	}
}

func (l *JSONLogger) TestResult(tt ethr.TestType, success bool, protocol ethr.Protocol, rIP net.IP, rPort uint16, result interface{}) {
	if l.active {
		resultJSON, _ := json.Marshal(NewTestResultLog(tt, success, protocol, rIP, rPort, result))
		l.toLog <- string(resultJSON)
	}
}
