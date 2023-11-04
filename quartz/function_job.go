package quartz

import (
	"context"
	"fmt"
	"sync"
)

// Function represents an argument-less function which returns
// a generic type R and a possible error.
type Function[R any] func(context.Context) (R, error)

// FunctionJob represents a Job that invokes the passed Function,
// implements the quartz.Job interface.
type FunctionJob[R any] struct {
	sync.RWMutex
	function  *Function[R]
	desc      string
	result    *R
	err       error
	jobStatus JobStatus
}

// NewFunctionJob returns a new FunctionJob without an explicit description.
func NewFunctionJob[R any](function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      fmt.Sprintf("FunctionJob:%p", &function),
		jobStatus: NA,
	}
}

// NewFunctionJobWithDesc returns a new FunctionJob with an explicit description.
func NewFunctionJobWithDesc[R any](desc string, function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      desc,
		jobStatus: NA,
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
	f.Lock()
	if err != nil {
		f.jobStatus = FAILURE
		f.result = nil
		f.err = err
	} else {
		f.jobStatus = OK
		f.result = &result
		f.err = nil
	}
	f.Unlock()
}

// Result returns the result of the FunctionJob.
func (f *FunctionJob[R]) Result() *R {
	f.RLock()
	defer f.RUnlock()
	return f.result
}

// Error returns the error of the FunctionJob.
func (f *FunctionJob[R]) Error() error {
	f.RLock()
	defer f.RUnlock()
	return f.err
}

// JobStatus returns the status of the FunctionJob.
func (f *FunctionJob[R]) JobStatus() JobStatus {
	f.RLock()
	defer f.RUnlock()
	return f.jobStatus
}
