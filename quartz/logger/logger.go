package logger

// A Logger handles log records.
type Logger interface {

	// Trace logs at LevelTrace.
	Trace(msg any)

	// Tracef logs at LevelTrace.
	Tracef(format string, args ...any)

	// Debug logs at LevelDebug.
	Debug(msg any)

	// Debugf logs at LevelDebug.
	Debugf(format string, args ...any)

	// Info logs at LevelInfo.
	Info(msg any)

	// Infof logs at LevelInfo.
	Infof(format string, args ...any)

	// Warn logs at LevelWarn.
	Warn(msg any)

	// Warnf logs at LevelWarn.
	Warnf(format string, args ...any)

	// Error logs at LevelError.
	Error(msg any)

	// Errorf logs at LevelError.
	Errorf(format string, args ...any)

	// Enabled reports whether the logger handles records at the given level.
	Enabled(level Level) bool
}
