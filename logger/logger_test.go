package logger_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
	l "github.com/reugn/go-quartz/logger"
)

func TestLogger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		supplier func(*bytes.Buffer, l.Level) l.Logger
	}{
		{
			name: "simple",
			supplier: func(b *bytes.Buffer, level l.Level) l.Logger {
				stdLogger := log.New(b, "", log.LstdFlags)
				return l.NewSimpleLogger(stdLogger, level)
			},
		},
		{
			name: "slog",
			supplier: func(b *bytes.Buffer, level l.Level) l.Logger {
				slogLogger := slog.New(slog.NewTextHandler(b, &slog.HandlerOptions{
					Level:     slog.Level(level),
					AddSource: true,
				}))
				var ctx context.Context // test nil context
				return l.NewSlogLogger(ctx, slogLogger)
			},
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			logger := test.supplier(&b, l.LevelInfo)

			logger.Trace("Trace")
			assertEmpty(t, &b)

			logger.Debug("Debug")
			assertEmpty(t, &b)

			logger.Info("Info")
			assertNotEmpty(t, &b)

			b.Reset()
			assertEmpty(t, &b)

			logger.Warn("Warn", "error", "err1")
			assertNotEmpty(t, &b)

			b.Reset()
			assertEmpty(t, &b)

			logger.Error("Error", "error")
			assertNotEmpty(t, &b)

			b.Reset()
			assertEmpty(t, &b)

			logger = test.supplier(&b, l.LevelTrace)

			logger.Trace("Trace")
			assertNotEmpty(t, &b)

			b.Reset()
			assertEmpty(t, &b)

			logger.Debug("Debug")
			assertNotEmpty(t, &b)

			b.Reset()
			assertEmpty(t, &b)

			logger = test.supplier(&b, l.LevelOff)

			logger.Error("Error")
			assertEmpty(t, &b)
		})
	}
}

func TestLogger_SlogPanic(t *testing.T) {
	assert.Panics(t, func() {
		l.NewSlogLogger(context.Background(), nil)
	})
}

func TestLogger_NoOp(_ *testing.T) {
	logger := l.NoOpLogger{}

	logger.Trace("Trace")
	logger.Debug("Debug")
	logger.Info("Info")
	logger.Warn("Warn")
	logger.Error("Error")
}

func assertEmpty(t *testing.T, r io.Reader) {
	t.Helper()
	logMsg := readAll(t, r)
	if logMsg != "" {
		t.Fatalf("log msg is not empty: %s", logMsg)
	}
}

func assertNotEmpty(t *testing.T, r io.Reader) {
	t.Helper()
	logMsg := readAll(t, r)
	if logMsg == "" {
		t.Fatal("log msg is empty")
	}
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
