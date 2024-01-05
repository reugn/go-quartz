package main

import (
	"context"
	"fmt"

	"github.com/reugn/go-quartz/quartz"
)

// PrintJob implements the quartz.Job interface.
type PrintJob struct {
	desc string
}

var _ quartz.Job = (*PrintJob)(nil)

// Description returns the description of the PrintJob.
func (pj *PrintJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *PrintJob) Execute(_ context.Context) error {
	fmt.Println("Executing " + pj.Description())
	return nil
}
