package main

import (
	"fmt"

	"github.com/reugn/go-quartz/quartz"
)

// PrintJob implements the quartz.Job interface.
type PrintJob struct {
	desc string
}

// Description returns the description of the PrintJob.
func (pj *PrintJob) Description() string {
	return pj.desc
}

// Key returns the unique PrintJob key.
func (pj *PrintJob) Key() int {
	return quartz.HashCode(pj.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *PrintJob) Execute() {
	fmt.Println("Executing " + pj.Description())
}
