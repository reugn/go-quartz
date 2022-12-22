package quartz

import (
	"context"
	"fmt"
)

// Function represents an argument-less function which returns a generic type R and a possible error.
type Function[R any] func(context.Context) (R, error)

// FunctionJob represents a Job that invokes the passed Function, implements the quartz.Job interface.
type FunctionJob[R any] struct {
	function  *Function[R]
	desc      string
	Result    *R
	Error     error
	JobStatus JobStatus
}

// NewFunctionJob returns a new FunctionJob without an explicit description.
func NewFunctionJob[R any](function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      fmt.Sprintf("FunctionJob:%p", &function),
		Result:    nil,
		Error:     nil,
		JobStatus: NA,
	}
}

// NewFunctionJob returns a new FunctionJob with an explicit description.
func NewFunctionJobWithDesc[R any](desc string, function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      desc,
		Result:    nil,
		Error:     nil,
		JobStatus: NA,
	}
}

// Description returns the description of the FunctionJob.
func (f *FunctionJob[R]) Description() string {
	return f.desc
}

// Key returns the unique FunctionJob key.
func (f *FunctionJob[R]) Key() int {
	return HashCode(fmt.Sprintf("%s:%p", f.desc, f.function))
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
// It invokes the held function, setting the results in Result and Error members.
func (f *FunctionJob[R]) Execute(ctx context.Context) {
	result, err := (*f.function)(ctx)
	if err != nil {
		f.JobStatus = FAILURE
		f.Result = nil
		f.Error = err
	} else {
		f.JobStatus = OK
		f.Error = nil
		f.Result = &result
	}
}
