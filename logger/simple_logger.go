package logger

import (
	"fmt"
	"log"
)

// SimpleLogger prefixes.
const (
	TracePrefix = "TRACE "
	DebugPrefix = "DEBUG "
	InfoPrefix  = "INFO "
	WarnPrefix  = "WARN "
	ErrorPrefix = "ERROR "
)

// SimpleLogger implements the logger.Logger interface.
type SimpleLogger struct {
	logger *log.Logger
	level  Level
}

var _ Logger = (*SimpleLogger)(nil)

// NewSimpleLogger returns a new SimpleLogger.
func NewSimpleLogger(logger *log.Logger, level Level) *SimpleLogger {
	return &SimpleLogger{
		logger: logger,
		level:  level,
	}
}

// Trace logs at LevelTrace.
// Arguments are handled in the manner of fmt.Println.
func (l *SimpleLogger) Trace(msg any) {
	if l.Enabled(LevelTrace) {
		l.logger.SetPrefix(TracePrefix)
		l.logger.Output(3, fmt.Sprint(msg))
	}
}

// Tracef logs at LevelTrace.
// Arguments are handled in the manner of fmt.Printf.
func (l *SimpleLogger) Tracef(format string, args ...any) {
	if l.Enabled(LevelTrace) {
		l.logger.SetPrefix(TracePrefix)
		l.logger.Output(3, fmt.Sprintf(format, args...))
	}
}

// Debug logs at LevelDebug.
// Arguments are handled in the manner of fmt.Println.
func (l *SimpleLogger) Debug(msg any) {
	if l.Enabled(LevelDebug) {
		l.logger.SetPrefix(DebugPrefix)
		l.logger.Output(3, fmt.Sprint(msg))
	}
}

// Debugf logs at LevelDebug.
// Arguments are handled in the manner of fmt.Printf.
func (l *SimpleLogger) Debugf(format string, args ...any) {
	if l.Enabled(LevelDebug) {
		l.logger.SetPrefix(DebugPrefix)
		l.logger.Output(3, fmt.Sprintf(format, args...))
	}
}

// Info logs at LevelInfo.
// Arguments are handled in the manner of fmt.Println.
func (l *SimpleLogger) Info(msg any) {
	if l.Enabled(LevelInfo) {
		l.logger.SetPrefix(InfoPrefix)
		l.logger.Output(3, fmt.Sprint(msg))
	}
}

// Infof logs at LevelInfo.
// Arguments are handled in the manner of fmt.Printf.
func (l *SimpleLogger) Infof(format string, args ...any) {
	if l.Enabled(LevelInfo) {
		l.logger.SetPrefix(InfoPrefix)
		l.logger.Output(3, fmt.Sprintf(format, args...))
	}
}

// Warn logs at LevelWarn.
// Arguments are handled in the manner of fmt.Println.
func (l *SimpleLogger) Warn(msg any) {
	if l.Enabled(LevelWarn) {
		l.logger.SetPrefix(WarnPrefix)
		l.logger.Output(3, fmt.Sprint(msg))
	}
}

// Warnf logs at LevelWarn.
// Arguments are handled in the manner of fmt.Printf.
func (l *SimpleLogger) Warnf(format string, args ...any) {
	if l.Enabled(LevelWarn) {
		l.logger.SetPrefix(WarnPrefix)
		l.logger.Output(3, fmt.Sprintf(format, args...))
	}
}

// Error logs at LevelError.
// Arguments are handled in the manner of fmt.Println.
func (l *SimpleLogger) Error(msg any) {
	if l.Enabled(LevelError) {
		l.logger.SetPrefix(ErrorPrefix)
		l.logger.Output(3, fmt.Sprint(msg))
	}
}

// Errorf logs at LevelError.
// Arguments are handled in the manner of fmt.Printf.
func (l *SimpleLogger) Errorf(format string, args ...any) {
	if l.Enabled(LevelError) {
		l.logger.SetPrefix(ErrorPrefix)
		l.logger.Output(3, fmt.Sprintf(format, args...))
	}
}

// Enabled reports whether the logger handles records at the given level.
func (l *SimpleLogger) Enabled(level Level) bool {
	return level >= l.level
}
