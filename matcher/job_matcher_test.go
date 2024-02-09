package matcher_test

import (
	"context"
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/matcher"
	"github.com/reugn/go-quartz/quartz"
)

func TestMatcher_JobAll(t *testing.T) {
	sched := quartz.NewStdScheduler()

	dummy := job.NewFunctionJob(func(_ context.Context) (bool, error) {
		return true, nil
	})
	cron, err := quartz.NewCronTrigger("@daily")
	assert.IsNil(t, err)

	jobKeys := []*quartz.JobKey{
		quartz.NewJobKey("job_monitor"),
		quartz.NewJobKey("job_update"),
		quartz.NewJobKeyWithGroup("job_monitor", "group_monitor"),
		quartz.NewJobKeyWithGroup("job_update", "group_update"),
	}

	for _, jobKey := range jobKeys {
		err := sched.ScheduleJob(quartz.NewJobDetail(dummy, jobKey), cron)
		assert.IsNil(t, err)
	}
	sched.Start(context.Background())

	assert.Equal(t, len(sched.GetJobKeys(matcher.JobActive())), 4)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobPaused())), 0)

	assert.Equal(t, len(sched.GetJobKeys(matcher.JobGroupEquals(quartz.DefaultGroup))), 2)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobGroupContains("_"))), 2)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobGroupStartsWith("group_"))), 2)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobGroupEndsWith("_update"))), 1)

	assert.Equal(t, len(sched.GetJobKeys(matcher.JobNameEquals("job_monitor"))), 2)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobNameContains("_"))), 4)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobNameStartsWith("job_"))), 4)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobNameEndsWith("_update"))), 2)

	// multiple matchers
	assert.Equal(t, len(sched.GetJobKeys(
		matcher.JobNameEquals("job_monitor"),
		matcher.JobGroupEquals(quartz.DefaultGroup),
		matcher.JobActive(),
	)), 1)

	assert.Equal(t, len(sched.GetJobKeys(
		matcher.JobNameEquals("job_monitor"),
		matcher.JobGroupEquals(quartz.DefaultGroup),
		matcher.JobPaused(),
	)), 0)

	// no matchers
	assert.Equal(t, len(sched.GetJobKeys()), 4)

	err = sched.PauseJob(quartz.NewJobKey("job_monitor"))
	assert.IsNil(t, err)

	assert.Equal(t, len(sched.GetJobKeys(matcher.JobActive())), 3)
	assert.Equal(t, len(sched.GetJobKeys(matcher.JobPaused())), 1)

	sched.Stop()
}

func TestMatcher_JobSwitchType(t *testing.T) {
	tests := []struct {
		name string
		m    quartz.Matcher[quartz.ScheduledJob]
	}{
		{
			name: "job-active",
			m:    matcher.JobActive(),
		},
		{
			name: "job-group-equals",
			m:    matcher.JobGroupEquals("group1"),
		},
		{
			name: "job-name-contains",
			m:    matcher.JobNameContains("name"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch jm := tt.m.(type) {
			case *matcher.JobStatus:
				assert.Equal(t, jm.Suspended, false)
			case *matcher.JobGroup:
				if jm.Operator != &matcher.StringEquals {
					t.Fatal("JobGroup unexpected operator")
				}
			case *matcher.JobName:
				if jm.Operator != &matcher.StringContains {
					t.Fatal("JobName unexpected operator")
				}
			default:
				t.Fatal("Unexpected matcher type")
			}
		})
	}
}

func TestMatcher_CustomStringOperator(t *testing.T) {
	var op matcher.StringOperator = func(_, _ string) bool { return true }
	assert.NotEqual(t, matcher.NewJobGroup(&op, "group1"), nil)
}
