package main

import (
	"fmt"

	"github.com/reugn/go-quartz/quartz"
)

//implements quartz.Job interface
type PrintJob struct {
	desc string
}

func (pj PrintJob) Description() string {
	return pj.desc
}

func (pj PrintJob) Key() int {
	return quartz.HashCode(pj.Description())
}

func (pj PrintJob) Execute() {
	fmt.Println("Executing " + pj.Description())
}
