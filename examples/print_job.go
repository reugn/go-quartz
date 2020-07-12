package main

import (
	"fmt"

	"github.com/reugn/go-quartz/quartz"
)

// PrintJob implements the quartz.Job interface.
type PrintJob struct {
	desc string
}

// Description returns a PrintJob description.
func (pj PrintJob) Description() string {
	return pj.desc
}

// Key returns a PrintJob unique key.
func (pj PrintJob) Key() int {
	return quartz.HashCode(pj.Description())
}

// Execute Called by the Scheduler when a Trigger fires that is associated with the Job.
func (pj PrintJob) Execute() {
	fmt.Println("Executing " + pj.Description())
}
