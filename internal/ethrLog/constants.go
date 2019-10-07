package ethrLog

// LogLevel specifies the logging level to use in both screen and
// file-based logging
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelDebug
)
