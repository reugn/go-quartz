package logger

// Logger is an interface for handling structured log records at different
// severity levels.
type Logger interface {
	Trace(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NoOpLogger satisfies the Logger interface and discards all log messages.
type NoOpLogger struct{}

var _ Logger = (*NoOpLogger)(nil)

func (NoOpLogger) Trace(_ string, _ ...any) {}
func (NoOpLogger) Debug(_ string, _ ...any) {}
func (NoOpLogger) Info(_ string, _ ...any)  {}
func (NoOpLogger) Warn(_ string, _ ...any)  {}
func (NoOpLogger) Error(_ string, _ ...any) {}
