package quartz

import "time"

// JobDetailOptions represents additional JobDetail properties.
type JobDetailOptions struct {
	// MaxRetries is the maximum number of retries before aborting the
	// current job execution.
	// Default: 0.
	MaxRetries int

	// RetryInterval is the fixed time interval between retry attempts.
	// Default: 1 second.
	RetryInterval time.Duration

	// Replace specifies whether the job should replace an existing job
	// with the same key.
	// Default: false.
	Replace bool
}

// NewDefaultJobDetailOptions returns a new instance of JobDetailOptions
// with the default values.
func NewDefaultJobDetailOptions() *JobDetailOptions {
	return &JobDetailOptions{
		MaxRetries:    0,
		RetryInterval: time.Second,
		Replace:       false,
	}
}

// JobDetail conveys the detail properties of a given Job instance.
type JobDetail struct {
	job    Job
	jobKey *JobKey
	opts   *JobDetailOptions
}

// NewJobDetail creates and returns a new JobDetail.
func NewJobDetail(job Job, jobKey *JobKey) *JobDetail {
	return NewJobDetailWithOptions(job, jobKey, NewDefaultJobDetailOptions())
}

// NewJobDetailWithOptions creates and returns a new JobDetail configured as specified.
func NewJobDetailWithOptions(job Job, jobKey *JobKey, opts *JobDetailOptions) *JobDetail {
	return &JobDetail{
		job:    job,
		jobKey: jobKey,
		opts:   opts,
	}
}

// Job returns job.
func (jd *JobDetail) Job() Job {
	return jd.job
}

// JobKey returns jobKey.
func (jd *JobDetail) JobKey() *JobKey {
	return jd.jobKey
}

// Options returns opts.
func (jd *JobDetail) Options() *JobDetailOptions {
	return jd.opts
}
