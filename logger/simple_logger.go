package logger

import (
	"fmt"
	"log"
	"strings"
)

// A Level is the importance or severity of a log event.
// The higher the level, the more important or severe the event.
type Level int

// Log levels.
const (
	LevelTrace Level = -8
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
	LevelOff   Level = 12
)

// SimpleLogger prefixes.
const (
	tracePrefix = "TRACE "
	debugPrefix = "DEBUG "
	infoPrefix  = "INFO "
	warnPrefix  = "WARN "
	errorPrefix = "ERROR "
)

// SimpleLogger implements the [Logger] interface.
type SimpleLogger struct {
	logger *log.Logger
	level  Level
}

var _ Logger = (*SimpleLogger)(nil)

// NewSimpleLogger returns a new [SimpleLogger].
func NewSimpleLogger(logger *log.Logger, level Level) *SimpleLogger {
	return &SimpleLogger{
		logger: logger,
		level:  level,
	}
}

// Trace logs at the trace level.
func (l *SimpleLogger) Trace(msg string, args ...any) {
	if l.enabled(LevelTrace) {
		l.logger.SetPrefix(tracePrefix)
		_ = l.logger.Output(2, formatMessage(msg, args))
	}
}

// Debug logs at the debug level.
func (l *SimpleLogger) Debug(msg string, args ...any) {
	if l.enabled(LevelDebug) {
		l.logger.SetPrefix(debugPrefix)
		_ = l.logger.Output(2, formatMessage(msg, args))
	}
}

// Info logs at the info level.
func (l *SimpleLogger) Info(msg string, args ...any) {
	if l.enabled(LevelInfo) {
		l.logger.SetPrefix(infoPrefix)
		_ = l.logger.Output(2, formatMessage(msg, args))
	}
}

// Warn logs at the warn level.
func (l *SimpleLogger) Warn(msg string, args ...any) {
	if l.enabled(LevelWarn) {
		l.logger.SetPrefix(warnPrefix)
		_ = l.logger.Output(2, formatMessage(msg, args))
	}
}

// Error logs at the error level.
func (l *SimpleLogger) Error(msg string, args ...any) {
	if l.enabled(LevelError) {
		l.logger.SetPrefix(errorPrefix)
		_ = l.logger.Output(2, formatMessage(msg, args))
	}
}

// enabled reports whether the SimpleLogger handles records at the given level.
func (l *SimpleLogger) enabled(level Level) bool {
	return level >= l.level
}

func formatMessage(msg string, args []any) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "msg=%s", msg)

	n := len(args)
	for i := 0; i < n; i += 2 {
		if i+1 < n {
			_, _ = fmt.Fprintf(&b, ", %s=%v", args[i], args[i+1])
		} else {
			_, _ = fmt.Fprintf(&b, ", %v", args[i])
		}
	}

	return b.String()
}
