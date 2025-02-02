package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create scheduler
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sched, _ := quartz.NewStdScheduler(quartz.WithLogger(logger.NewSlogLogger(ctx, slogLogger)))

	// async start scheduler
	sched.Start(ctx)

	// create jobs
	cronTrigger, _ := quartz.NewCronTrigger("1/5 * * * * *")
	shellJob := job.NewShellJob("ls -la")

	request, _ := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	curlJob := job.NewCurlJob(request)

	functionJob := job.NewFunctionJob(func(_ context.Context) (int, error) { return 42, nil })

	// register jobs to scheduler
	_ = sched.ScheduleJob(quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob")),
		cronTrigger)
	_ = sched.ScheduleJob(quartz.NewJobDetail(curlJob, quartz.NewJobKey("curlJob")),
		quartz.NewSimpleTrigger(7*time.Second))
	_ = sched.ScheduleJob(quartz.NewJobDetail(functionJob, quartz.NewJobKey("functionJob")),
		quartz.NewSimpleTrigger(5*time.Second))

	// stop scheduler
	sched.Stop()

	// wait for all workers to exit
	sched.Wait(ctx)
}
