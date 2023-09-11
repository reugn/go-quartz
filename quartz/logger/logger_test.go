package logger_test

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/reugn/go-quartz/quartz/logger"
)

func TestSimpleLogger(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger.SetDefault(logger.NewSimpleLogger(stdLogger, logger.LevelInfo))

	logger.Trace("Trace")
	assertEmpty(&b, t)
	logger.Tracef("Trace%s", "f")
	assertEmpty(&b, t)

	logger.Debug("Debug")
	assertEmpty(&b, t)
	logger.Debugf("Debug%s", "f")
	assertEmpty(&b, t)

	logger.Info("Info")
	assertNotEmpty(&b, t)
	logger.Infof("Info%s", "f")
	assertNotEmpty(&b, t)

	logger.Warn("Warn")
	assertNotEmpty(&b, t)
	logger.Warnf("Warn%s", "f")
	assertNotEmpty(&b, t)

	logger.Error("Error")
	assertNotEmpty(&b, t)
	logger.Errorf("Error%s", "f")
	assertNotEmpty(&b, t)
}

func TestLoggerOff(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger.SetDefault(logger.NewSimpleLogger(stdLogger, logger.LevelOff))

	logger.Error("Error")
	assertEmpty(&b, t)
	logger.Errorf("Error%s", "f")
	assertEmpty(&b, t)
}

func TestLogFormat(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logr := logger.NewSimpleLogger(stdLogger, logger.LevelTrace)
	logger.SetDefault(logr)

	empty := struct{}{}
	logr.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)

	logr.Debugf("Debugf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Debugf("Debugf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)

	logr.Infof("Infof: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Infof("Infof: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)

	logr.Warnf("Warnf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Warnf("Warnf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)

	logr.Errorf("Errorf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Errorf("Errorf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
}

func assertEmpty(r io.Reader, t *testing.T) {
	logMsg := readAll(r, t)
	if logMsg != "" {
		t.Fatalf("Log msg is not empty: %s", logMsg)
	}
}

func assertNotEmpty(r io.Reader, t *testing.T) {
	logMsg := readAll(r, t)
	if logMsg == "" {
		t.Fatal("Log msg is empty")
	}
}

func checkLogFormat(r io.Reader, t *testing.T) {
	logMsg := readAll(r, t)
	if !strings.Contains(logMsg, "a, 1, true, {}") {
		t.Fatalf("Invalid log format: %s", logMsg)
	}
}

func readAll(r io.Reader, t *testing.T) string {
	bytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}
