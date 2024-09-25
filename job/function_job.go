package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/reugn/go-quartz/quartz"
)

// Function represents a function which takes a [context.Context] as its
// only argument and returns a generic type R and a possible error.
type Function[R any] func(context.Context) (R, error)

// FunctionJob represents a Job that invokes the passed [Function],
// implements the [quartz.Job] interface.
type FunctionJob[R any] struct {
	mtx         sync.RWMutex
	function    Function[R]
	description string
	result      R
	err         error
	jobStatus   Status
}

var _ quartz.Job = (*FunctionJob[any])(nil)

// NewFunctionJob returns a new [FunctionJob] with a generated description.
func NewFunctionJob[R any](function Function[R]) *FunctionJob[R] {
	return NewFunctionJobWithDesc(
		function,
		fmt.Sprintf("FunctionJob%s%p", quartz.Sep, &function),
	)
}

// NewFunctionJobWithDesc returns a new [FunctionJob] with the specified
// description.
func NewFunctionJobWithDesc[R any](function Function[R],
	description string) *FunctionJob[R] {
	return &FunctionJob[R]{
		function:    function,
		description: description,
		jobStatus:   StatusNA,
	}
}

// Description returns the description of the FunctionJob.
func (f *FunctionJob[R]) Description() string {
	return f.description
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
// It invokes the held function, setting the results in result and error members.
func (f *FunctionJob[R]) Execute(ctx context.Context) error {
	result, err := f.function(ctx)
	f.mtx.Lock()
	if err != nil {
		var zero R
		f.jobStatus = StatusFailure
		f.result, f.err = zero, err
	} else {
		f.jobStatus = StatusOK
		f.result, f.err = result, nil
	}
	f.mtx.Unlock()
	return err
}

// Result returns the result of the FunctionJob.
func (f *FunctionJob[R]) Result() R {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.result
}

// Error returns the error of the FunctionJob.
func (f *FunctionJob[R]) Error() error {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.err
}

// JobStatus returns the status of the FunctionJob.
func (f *FunctionJob[R]) JobStatus() Status {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.jobStatus
}
