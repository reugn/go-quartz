package main

import (
	"context"
	"net/http"
	"time"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create scheduler
	sched := quartz.NewStdScheduler()

	// async start scheduler
	sched.Start(ctx)

	// create jobs
	cronTrigger, _ := quartz.NewCronTrigger("1/5 * * * * *")
	shellJob := job.NewShellJob("ls -la")

	request, _ := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	curlJob := job.NewCurlJob(request)

	functionJob := job.NewFunctionJob(func(_ context.Context) (int, error) { return 42, nil })

	// register jobs to scheduler
	sched.ScheduleJob(quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob")),
		cronTrigger)
	sched.ScheduleJob(quartz.NewJobDetail(curlJob, quartz.NewJobKey("curlJob")),
		quartz.NewSimpleTrigger(time.Second*7))
	sched.ScheduleJob(quartz.NewJobDetail(functionJob, quartz.NewJobKey("functionJob")),
		quartz.NewSimpleTrigger(time.Second*5))

	// stop scheduler
	sched.Stop()

	// wait for all workers to exit
	sched.Wait(ctx)
}
