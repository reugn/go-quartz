package quartz_test

import (
	"bytes"
)

type BufferedLogger struct {
	buffer *bytes.Buffer
}

func (b *BufferedLogger) Log(msg string) {
	b.buffer.WriteString(msg + "\n")
}

func (b *BufferedLogger) GetContents() string {
	return b.buffer.String()
}

func NewBufferedLogger() *BufferedLogger {
	return &BufferedLogger{
		buffer: new(bytes.Buffer),
	}
}
