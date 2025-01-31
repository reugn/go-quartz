package logger_test

import (
	"bytes"
	"io"
	"log"
	"testing"

	l "github.com/reugn/go-quartz/logger"
)

func TestLogger_Simple(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger := l.NewSimpleLogger(stdLogger, l.LevelInfo)

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

	logger = l.NewSimpleLogger(stdLogger, l.LevelTrace)

	logger.Trace("Trace")
	assertNotEmpty(t, &b)

	b.Reset()
	assertEmpty(t, &b)

	logger.Debug("Debug")
	assertNotEmpty(t, &b)
}

func TestLogger_SimpleOff(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger := l.NewSimpleLogger(stdLogger, l.LevelOff)

	logger.Error("Error")
	assertEmpty(t, &b)
}

func TestLogger_NoOp(t *testing.T) {
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
