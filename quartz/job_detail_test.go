package quartz_test

import (
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func TestJobDetail(t *testing.T) {
	job := job.NewShellJob("ls -la")
	jobKey := quartz.NewJobKey("job")
	jobDetail := quartz.NewJobDetail(job, jobKey)

	assert.Equal(t, jobDetail.Job(), quartz.Job(job))
	assert.Equal(t, jobDetail.JobKey(), jobKey)
	assert.Equal(t, jobDetail.Options(), quartz.NewDefaultJobDetailOptions())
}

func TestJobDetailWithOptions(t *testing.T) {
	job := job.NewShellJob("ls -la")
	jobKey := quartz.NewJobKey("job")
	opts := quartz.NewDefaultJobDetailOptions()
	opts.MaxRetries = 3
	opts.RetryInterval = 100 * time.Millisecond
	jobDetail := quartz.NewJobDetailWithOptions(job, jobKey, opts)

	assert.Equal(t, jobDetail.Job(), quartz.Job(job))
	assert.Equal(t, jobDetail.JobKey(), jobKey)
	assert.Equal(t, jobDetail.Options(), opts)
}
