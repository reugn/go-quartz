package logger

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// SlogLogger implements the [Logger] interface, providing a structured logging
// mechanism by delegating log operations to the standard library's slog package.
//
// In addition to the default slog levels, it introduces the Trace (Debug-4) level.
// To change the string representation of the Trace log level, use the ReplaceAttr
// field in [slog.HandlerOptions], e.g.
//
//	myLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
//		Level:     slog.Level(logger.LevelTrace),
//		AddSource: true,
//		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
//			if a.Key == slog.LevelKey {
//				level := a.Value.Any().(slog.Level)
//				if level == slog.Level(logger.LevelTrace) {
//					a.Value = slog.StringValue("TRACE")
//				}
//			}
//			return a
//		},
//	}))
type SlogLogger struct {
	ctx    context.Context
	logger *slog.Logger
}

var _ Logger = (*SlogLogger)(nil)

// NewSlogLogger returns a new [SlogLogger].
// It will panic if the logger is nil.
func NewSlogLogger(ctx context.Context, logger *slog.Logger) *SlogLogger {
	if logger == nil {
		panic("nil logger")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &SlogLogger{
		ctx:    ctx,
		logger: logger,
	}
}

// Trace logs at the trace level.
func (l *SlogLogger) Trace(msg string, args ...any) {
	l.log(slog.Level(LevelTrace), msg, args...)
}

// Debug logs at the debug level.
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.log(slog.LevelDebug, msg, args...)
}

// Info logs at the info level.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.log(slog.LevelInfo, msg, args...)
}

// Warn logs at the warn level.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.log(slog.LevelWarn, msg, args...)
}

// Error logs at the error level.
func (l *SlogLogger) Error(msg string, args ...any) {
	l.log(slog.LevelError, msg, args...)
}

// log is the low-level logging method, obtaining the caller's PC for context.
func (l *SlogLogger) log(level slog.Level, msg string, args ...any) {
	if !l.logger.Enabled(l.ctx, level) {
		return
	}

	// skip [runtime.Callers, this function, this function's caller]
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	pc := pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)

	_ = l.logger.Handler().Handle(l.ctx, r)
}
