//nolint:dupl
package matcher

import (
	"github.com/reugn/go-quartz/quartz"
)

// JobGroup implements the quartz.Matcher interface with the type argument
// quartz.ScheduledJob, matching jobs by their group name.
// It has public fields to allow predicate pushdown in custom quartz.JobQueue
// implementations.
type JobGroup struct {
	Operator *StringOperator // uses a pointer to compare with standard operators
	Pattern  string
}

var _ quartz.Matcher[quartz.ScheduledJob] = (*JobGroup)(nil)

// NewJobGroup returns a new JobGroup matcher given the string operator and pattern.
func NewJobGroup(operator *StringOperator, pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return &JobGroup{
		Operator: operator,
		Pattern:  pattern,
	}
}

// JobGroupEquals returns a new JobGroup, matching jobs whose group name is identical
// to the given string pattern.
func JobGroupEquals(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobGroup(&StringEquals, pattern)
}

// JobGroupStartsWith returns a new JobGroup, matching jobs whose group name starts
// with the given string pattern.
func JobGroupStartsWith(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobGroup(&StringStartsWith, pattern)
}

// JobGroupEndsWith returns a new JobGroup, matching jobs whose group name ends
// with the given string pattern.
func JobGroupEndsWith(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobGroup(&StringEndsWith, pattern)
}

// JobGroupContains returns a new JobGroup, matching jobs whose group name contains
// the given string pattern.
func JobGroupContains(pattern string) quartz.Matcher[quartz.ScheduledJob] {
	return NewJobGroup(&StringContains, pattern)
}

// IsMatch evaluates JobGroup matcher on the given job.
func (g *JobGroup) IsMatch(job quartz.ScheduledJob) bool {
	return (*g.Operator)(job.JobDetail().JobKey().Group(), g.Pattern)
}
