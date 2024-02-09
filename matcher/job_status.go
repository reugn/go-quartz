package matcher

import (
	"github.com/reugn/go-quartz/quartz"
)

// JobStatus implements the quartz.Matcher interface with the type argument
// quartz.ScheduledJob, matching jobs by their status.
// It has public fields to allow predicate pushdown in custom quartz.JobQueue
// implementations.
type JobStatus struct {
	Suspended bool
}

var _ quartz.Matcher[quartz.ScheduledJob] = (*JobStatus)(nil)

// JobActive returns a matcher to match active jobs.
func JobActive() quartz.Matcher[quartz.ScheduledJob] {
	return &JobStatus{false}
}

// JobPaused returns a matcher to match paused jobs.
func JobPaused() quartz.Matcher[quartz.ScheduledJob] {
	return &JobStatus{true}
}

// IsMatch evaluates JobStatus matcher on the given job.
func (s *JobStatus) IsMatch(job quartz.ScheduledJob) bool {
	return job.JobDetail().Options().Suspended == s.Suspended
}
