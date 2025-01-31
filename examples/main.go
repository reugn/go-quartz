package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		<-sigch
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go sampleJobs(ctx, &wg)
	go sampleScheduler(ctx, &wg)

	wg.Wait()
}

func sampleScheduler(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	stdLogger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix|log.Lshortfile)
	l := logger.NewSimpleLogger(stdLogger, logger.LevelInfo)
	sched, err := quartz.NewStdScheduler(quartz.WithLogger(l))
	if err != nil {
		fmt.Println(err)
		return
	}

	cronTrigger, err := quartz.NewCronTrigger("1/3 * * * * *")
	if err != nil {
		fmt.Println(err)
		return
	}

	cronJob := quartz.NewJobDetail(&PrintJob{"Cron job"}, quartz.NewJobKey("cronJob"))
	sched.Start(ctx)

	runOnceJobDetail := quartz.NewJobDetail(&PrintJob{"Ad hoc Job"}, quartz.NewJobKey("runOnceJob"))
	jobDetail1 := quartz.NewJobDetail(&PrintJob{"First job"}, quartz.NewJobKey("job1"))
	jobDetail2 := quartz.NewJobDetail(&PrintJob{"Second job"}, quartz.NewJobKey("job2"))
	jobDetail3 := quartz.NewJobDetail(&PrintJob{"Third job"}, quartz.NewJobKey("job3"))
	_ = sched.ScheduleJob(runOnceJobDetail, quartz.NewRunOnceTrigger(5*time.Second))
	_ = sched.ScheduleJob(jobDetail1, quartz.NewSimpleTrigger(12*time.Second))
	_ = sched.ScheduleJob(jobDetail2, quartz.NewSimpleTrigger(6*time.Second))
	_ = sched.ScheduleJob(jobDetail3, quartz.NewSimpleTrigger(3*time.Second))
	_ = sched.ScheduleJob(cronJob, cronTrigger)

	time.Sleep(10 * time.Second)

	scheduledJob, err := sched.GetScheduledJob(cronJob.JobKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(scheduledJob.Trigger().Description())
	jobKeys, _ := sched.GetJobKeys()
	fmt.Println("Before delete: ", jobKeys)
	_ = sched.DeleteJob(cronJob.JobKey())
	jobKeys, _ = sched.GetJobKeys()
	fmt.Println("After delete: ", jobKeys)

	time.Sleep(2 * time.Second)

	sched.Stop()
	sched.Wait(ctx)
}

func sampleJobs(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	sched, err := quartz.NewStdScheduler()
	if err != nil {
		fmt.Println(err)
		return
	}

	sched.Start(ctx)

	cronTrigger, err := quartz.NewCronTrigger("1/5 * * * * *")
	if err != nil {
		fmt.Println(err)
		return
	}

	shellJob := job.NewShellJob("ls -la")
	request, err := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	curlJob := job.NewCurlJob(request)
	functionJob := job.NewFunctionJobWithDesc(
		func(_ context.Context) (int, error) { return 42, nil },
		"42")

	shellJobDetail := quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob"))
	curlJobDetail := quartz.NewJobDetail(curlJob, quartz.NewJobKey("curlJob"))
	functionJobDetail := quartz.NewJobDetail(functionJob, quartz.NewJobKey("functionJob"))
	_ = sched.ScheduleJob(shellJobDetail, cronTrigger)
	_ = sched.ScheduleJob(curlJobDetail, quartz.NewSimpleTrigger(7*time.Second))
	_ = sched.ScheduleJob(functionJobDetail, quartz.NewSimpleTrigger(3*time.Second))

	time.Sleep(10 * time.Second)

	fmt.Println(sched.GetJobKeys())
	fmt.Println(shellJob.Stdout())

	response, err := curlJob.DumpResponse(true)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(response))
	}
	fmt.Printf("Function job result: %v\n", functionJob.Result())

	time.Sleep(2 * time.Second)

	sched.Stop()
	sched.Wait(ctx)
}
