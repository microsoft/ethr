//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package ethrLog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/microsoft/ethr/utils"
)

type logMessage struct {
	Time    string
	Type    string
	Message string
}

type logLatencyData struct {
	Time       string
	Type       string
	RemoteAddr string
	Protocol   string
	Avg        string
	Min        string
	P50        string
	P90        string
	P95        string
	P99        string
	P999       string
	P9999      string
	Max        string
}

type logTestResults struct {
	Time                 string
	Type                 string
	RemoteAddr           string
	Protocol             string
	BitsPerSecond        string
	ConnectionsPerSecond string
	PacketsPerSecond     string
	AverageLatency       string
}

var loggingActive = false
var logChan = make(chan string, 64)

func LogInit(fileName string) {
	if fileName == "" {
		return
	}
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("Unable to open the log file %s, Error: %v", fileName, err)
		return
	}
	log.SetFlags(0)
	log.SetOutput(logFile)
	loggingActive = true
	go runLogger(logFile)
}

func LogFini() {
	loggingActive = false
}

func runLogger(logFile *os.File) {
	for loggingActive {
		s := <-logChan
		log.Println(s)
	}
	logFile.Close()
}

func logMsg(prefix, msg string) {
	if loggingActive {
		logData := logMessage{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = prefix
		logData.Message = msg
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}

func Info(msg string) {
	logMsg("INFO", msg)
}

func Error(msg string) {
	logMsg("ERROR", msg)
}

func Debug(msg string) {
	logMsg("DEBUG", msg)
}

func LogResults(s []string) {
	if loggingActive {
		logData := logTestResults{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = "TestResult"
		logData.RemoteAddr = s[0]
		logData.Protocol = s[1]
		logData.BitsPerSecond = s[2]
		logData.ConnectionsPerSecond = s[3]
		logData.PacketsPerSecond = s[4]
		logData.AverageLatency = s[5]
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}

func LogLatency(remoteAddr, proto string, avg, min, p50, p90, p95, p99, p999, p9999, max time.Duration) {
	if loggingActive {
		logData := logLatencyData{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = "LatencyResult"
		logData.RemoteAddr = remoteAddr
		logData.Protocol = proto
		logData.Avg = utils.DurationToString(avg)
		logData.Min = utils.DurationToString(min)
		logData.P50 = utils.DurationToString(p50)
		logData.P90 = utils.DurationToString(p90)
		logData.P95 = utils.DurationToString(p95)
		logData.P99 = utils.DurationToString(p99)
		logData.P999 = utils.DurationToString(p999)
		logData.P9999 = utils.DurationToString(p9999)
		logData.Max = utils.DurationToString(max)
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}
