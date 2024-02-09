//nolint:dupl
package matcher

import (
	"github.com/reugn/go-quartz/quartz"
)

// JobName implements the quartz.Matcher interface with the type argument
// quartz.ScheduledJob, matching jobs by their name.
// It has public fields to allow predicate pushdown in custom quartz.JobQueue
// implementations.
type JobName struct {
	Operator *StringOperator // uses a pointer to compare with standard operators
	Pattern  string
}

var _ quartz.Matcher[quartz.ScheduledJob] = (*JobName)(nil)

// NewJobName returns a new JobName matcher given the string operator and pattern.
func NewJobName(operator *StringOperator, pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return &JobName{
		Operator: operator,
		Pattern:  pattern,
	}
}

// JobNameEquals returns a new JobName, matching jobs whose name is identical
// to the given string pattern.
func JobNameEquals(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobName(&StringEquals, pattern)
}

// JobNameStartsWith returns a new JobName, matching jobs whose name starts
// with the given string pattern.
func JobNameStartsWith(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobName(&StringStartsWith, pattern)
}

// JobNameEndsWith returns a new JobName, matching jobs whose name ends
// with the given string pattern.
func JobNameEndsWith(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobName(&StringEndsWith, pattern)
}

// JobNameContains returns a new JobName, matching jobs whose name contains
// the given string pattern.
func JobNameContains(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobName(&StringContains, pattern)
}

// IsMatch evaluates JobName matcher on the given job.
func (n *JobName) IsMatch(job quartz.ScheduledJob) bool {
	return (*n.Operator)(job.JobDetail().JobKey().Name(), n.Pattern)
}
