package logger_test

import (
	"bytes"
	"io"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/reugn/go-quartz/logger"
)

func TestSimpleLogger(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger.SetDefault(logger.NewSimpleLogger(stdLogger, logger.LevelInfo))

	logger.Trace("Trace")
	assertEmpty(t, &b)
	logger.Tracef("Trace%s", "f")
	assertEmpty(t, &b)

	logger.Debug("Debug")
	assertEmpty(t, &b)
	logger.Debugf("Debug%s", "f")
	assertEmpty(t, &b)

	logger.Info("Info")
	assertNotEmpty(t, &b)
	logger.Infof("Info%s", "f")
	assertNotEmpty(t, &b)

	logger.Warn("Warn")
	assertNotEmpty(t, &b)
	logger.Warnf("Warn%s", "f")
	assertNotEmpty(t, &b)

	logger.Error("Error")
	assertNotEmpty(t, &b)
	logger.Errorf("Error%s", "f")
	assertNotEmpty(t, &b)
}

func TestLoggerOff(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logger.SetDefault(logger.NewSimpleLogger(stdLogger, logger.LevelOff))

	if logger.Enabled(logger.LevelError) {
		t.Fatal("logger.LevelError is enabled")
	}
	logger.Error("Error")
	assertEmpty(t, &b)
	logger.Errorf("Error%s", "f")
	assertEmpty(t, &b)
}

func TestLoggerRace(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)

	logger1 := logger.NewSimpleLogger(stdLogger, logger.LevelOff)
	logger2 := logger.NewSimpleLogger(stdLogger, logger.LevelTrace)
	logger3 := logger.NewSimpleLogger(stdLogger, logger.LevelDebug)

	wg := sync.WaitGroup{}
	wg.Add(3)
	go setLogger(&wg, logger1)
	go setLogger(&wg, logger2)
	go setLogger(&wg, logger3)
	wg.Wait()
	wg.Add(1)
	go setLogger(&wg, logger2)
	wg.Wait()

	if logger.Default() != logger2 {
		t.Fatal("logger set race error")
	}
}

func setLogger(wg *sync.WaitGroup, l *logger.SimpleLogger) {
	defer wg.Done()
	logger.SetDefault(l)
}

func TestCustomLogger(t *testing.T) {
	l := &countingLogger{}
	logger.SetDefault(l)
	logger.Debug("debug")
	logger.Info("info")
	logger.Error("error")
	if l.Count != 3 {
		t.Fatal("custom logger error")
	}
}

func TestLogFormat(t *testing.T) {
	var b bytes.Buffer
	stdLogger := log.New(&b, "", log.LstdFlags)
	logr := logger.NewSimpleLogger(stdLogger, logger.LevelTrace)
	logger.SetDefault(logr)

	empty := struct{}{}
	logr.Trace("Trace")
	assertNotEmpty(t, &b)
	logr.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
	logger.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)

	logr.Debug("Debug")
	assertNotEmpty(t, &b)
	logr.Debugf("Debugf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
	logger.Debugf("Debugf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)

	logr.Infof("Infof: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
	logger.Infof("Infof: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)

	logr.Warnf("Warnf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
	logger.Warnf("Warnf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)

	logr.Errorf("Errorf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
	logger.Errorf("Errorf: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(t, &b)
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

func checkLogFormat(t *testing.T, r io.Reader) {
	t.Helper()
	logMsg := readAll(t, r)
	if !strings.Contains(logMsg, "a, 1, true, {}") {
		t.Fatalf("invalid log format: %s", logMsg)
	}
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	bytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}

type countingLogger struct {
	Count int
}

var _ logger.Logger = (*countingLogger)(nil)

func (l *countingLogger) Trace(_ any) {
	l.Count++
}

func (l *countingLogger) Tracef(_ string, _ ...any) {
	l.Count++
}

func (l *countingLogger) Debug(_ any) {
	l.Count++
}

func (l *countingLogger) Debugf(_ string, _ ...any) {
	l.Count++
}

func (l *countingLogger) Info(_ any) {
	l.Count++
}

func (l *countingLogger) Infof(_ string, _ ...any) {
	l.Count++
}

func (l *countingLogger) Warn(_ any) {
	l.Count++
}

func (l *countingLogger) Warnf(_ string, _ ...any) {
	l.Count++
}

func (l *countingLogger) Error(_ any) {
	l.Count++
}

func (l *countingLogger) Errorf(_ string, _ ...any) {
	l.Count++
}

func (l *countingLogger) Enabled(_ logger.Level) bool {
	return true
}
