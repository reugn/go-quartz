package quartz

import (
	"log"
)

type LoggerAdapter interface {
	Log(msg string)
}

type StdoutLogger struct {
}

func (s *StdoutLogger) Log(msg string) {
	log.Println(msg)
}
