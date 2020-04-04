package quartz_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestScheduler(t *testing.T) {
	sched := quartz.NewStdScheduler()
	shellJob := quartz.NewShellJob("ls -la")
	curlJob, err := quartz.NewCurlJob(http.MethodGet, "http://worldclockapi.com/api/json/est/now", "", nil)
	assertEqual(t, err, nil)
	sched.Start()
	sched.ScheduleJob(shellJob, quartz.NewRunOnceTrigger(time.Microsecond))
	sched.ScheduleJob(curlJob, quartz.NewRunOnceTrigger(time.Microsecond))
	time.Sleep(time.Second)
	sched.Stop()
	assertEqual(t, shellJob.JobStatus, quartz.OK)
	assertEqual(t, curlJob.JobStatus, quartz.OK)
}
