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
	sched.ScheduleJob(shellJob, quartz.NewSimpleTrigger(time.Millisecond*800))
	sched.ScheduleJob(curlJob, quartz.NewRunOnceTrigger(time.Millisecond))

	time.Sleep(time.Second)
	scheduledJobKeys := sched.GetJobKeys()
	assertEqual(t, scheduledJobKeys, []int{3059422767})

	_, err = sched.GetScheduledJob(jobKeys[0])
	if err != nil {
		t.Fail()
	}

	err = sched.DeleteJob(shellJob.Key())
	if err != nil {
		t.Fail()
	}
	assertEqual(t, sched.Queue.Len(), 0)

	sched.Clear()
	sched.Stop()
	assertEqual(t, shellJob.JobStatus, quartz.OK)
	assertEqual(t, curlJob.JobStatus, quartz.OK)
}
