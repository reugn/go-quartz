package job

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/reugn/go-quartz/quartz"
)

type isolatedJob struct {
	quartz.Job
	isRunning atomic.Bool
}

var _ quartz.Job = (*isolatedJob)(nil)

// Execute is called by a Scheduler when the Trigger associated
// with this job fires.
func (j *isolatedJob) Execute(ctx context.Context) error {
	if wasRunning := j.isRunning.Swap(true); wasRunning {
		return errors.New("job is running")
	}
	defer j.isRunning.Store(false)

	return j.Job.Execute(ctx)
}

// NewIsolatedJob wraps a job object and ensures that only one
// instance of the job's Execute method can be called at a time.
func NewIsolatedJob(underlying quartz.Job) quartz.Job {
	return &isolatedJob{
		Job: underlying,
	}
}
