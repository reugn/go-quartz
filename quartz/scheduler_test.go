package quartz_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestScheduler(t *testing.T) {
	sched := quartz.NewStdScheduler()
	var jobKeys [2]int

	shellJob := quartz.NewShellJob("ls -la")
	shellJob.Description()
	jobKeys[0] = shellJob.Key()

	curlJob, err := quartz.NewCurlJob(http.MethodGet, "http://worldclockapi.com/api/json/est/now", "", nil)
	assertEqual(t, err, nil)
	curlJob.Description()
	jobKeys[1] = curlJob.Key()

	sched.Start()
	sched.ScheduleJob(shellJob, quartz.NewRunOnceTrigger(time.Microsecond))
	sched.ScheduleJob(curlJob, quartz.NewRunOnceTrigger(time.Microsecond))

	time.Sleep(time.Second)
	scheduledJobKeys := sched.GetJobKeys()
	assertEqual(t, scheduledJobKeys, []int{})

	_, err = sched.GetScheduledJob(jobKeys[0])
	if err == nil {
		t.Fail()
	}
	err = sched.DeleteJob(jobKeys[0])
	if err == nil {
		t.Fail()
	}

	sched.Clear()
	sched.Stop()
	assertEqual(t, shellJob.JobStatus, quartz.OK)
	assertEqual(t, curlJob.JobStatus, quartz.OK)
}
