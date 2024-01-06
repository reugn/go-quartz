package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/reugn/go-quartz/quartz"
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
	jobStatus Status
}

var _ quartz.Job = (*FunctionJob[any])(nil)

// NewFunctionJob returns a new FunctionJob without an explicit description.
func NewFunctionJob[R any](function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      fmt.Sprintf("FunctionJob%s%p", quartz.Sep, &function),
		jobStatus: StatusNA,
	}
}

// NewFunctionJobWithDesc returns a new FunctionJob with an explicit description.
func NewFunctionJobWithDesc[R any](desc string, function Function[R]) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:  &function,
		desc:      desc,
		jobStatus: StatusNA,
	}
}

// Description returns the description of the FunctionJob.
func (f *FunctionJob[R]) Description() string {
	return f.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
// It invokes the held function, setting the results in Result and Error members.
func (f *FunctionJob[R]) Execute(ctx context.Context) error {
	result, err := (*f.function)(ctx)
	f.Lock()
	if err != nil {
		f.jobStatus = StatusFailure
		f.result = nil
		f.err = err
	} else {
		f.jobStatus = StatusOK
		f.result = &result
		f.err = nil
	}
	f.Unlock()
	return err
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
func (f *FunctionJob[R]) JobStatus() Status {
	f.RLock()
	defer f.RUnlock()
	return f.jobStatus
}
