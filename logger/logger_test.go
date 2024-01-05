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

	if logger.Enabled(logger.LevelError) {
		t.Fatal("logger.LevelError is enabled")
	}
	logger.Error("Error")
	assertEmpty(&b, t)
	logger.Errorf("Error%s", "f")
	assertEmpty(&b, t)
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
	assertNotEmpty(&b, t)
	logr.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)
	logger.Tracef("Tracef: %s, %d, %v, %v", "a", 1, true, empty)
	checkLogFormat(&b, t)

	logr.Debug("Debug")
	assertNotEmpty(&b, t)
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
		t.Fatalf("log msg is not empty: %s", logMsg)
	}
}

func assertNotEmpty(r io.Reader, t *testing.T) {
	logMsg := readAll(r, t)
	if logMsg == "" {
		t.Fatal("log msg is empty")
	}
}

func checkLogFormat(r io.Reader, t *testing.T) {
	logMsg := readAll(r, t)
	if !strings.Contains(logMsg, "a, 1, true, {}") {
		t.Fatalf("invalid log format: %s", logMsg)
	}
}

func readAll(r io.Reader, t *testing.T) string {
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
