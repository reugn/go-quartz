package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go sampleJobs(ctx, wg)
	go sampleScheduler(ctx, wg)

	wg.Wait()
}

func sampleScheduler(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	sched := quartz.NewStdScheduler()
	cronTrigger, err := quartz.NewCronTrigger("1/3 * * * * *")
	if err != nil {
		fmt.Println(err)
		return
	}

	cronJob := PrintJob{"Cron job"}
	sched.Start(ctx)

	sched.ScheduleJob(ctx, &PrintJob{"Ad hoc Job"}, quartz.NewRunOnceTrigger(time.Second*5))
	sched.ScheduleJob(ctx, &PrintJob{"First job"}, quartz.NewSimpleTrigger(time.Second*12))
	sched.ScheduleJob(ctx, &PrintJob{"Second job"}, quartz.NewSimpleTrigger(time.Second*6))
	sched.ScheduleJob(ctx, &PrintJob{"Third job"}, quartz.NewSimpleTrigger(time.Second*3))
	sched.ScheduleJob(ctx, &cronJob, cronTrigger)

	time.Sleep(time.Second * 10)

	scheduledJob, err := sched.GetScheduledJob(cronJob.Key())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(scheduledJob.TriggerDescription)
	fmt.Println("Before delete: ", sched.GetJobKeys())
	sched.DeleteJob(cronJob.Key())
	fmt.Println("After delete: ", sched.GetJobKeys())

	time.Sleep(time.Second * 2)
	sched.Stop()
	sched.Wait(ctx)
}

func sampleJobs(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	sched := quartz.NewStdScheduler()
	sched.Start(ctx)

	cronTrigger, err := quartz.NewCronTrigger("1/5 * * * * *")
	if err != nil {
		fmt.Println(err)
		return
	}

	shellJob := quartz.NewShellJob("ls -la")
	request, err := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	curlJob, err := quartz.NewCurlJob(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	functionJob := quartz.NewFunctionJobWithDesc("42", func(_ context.Context) (int, error) { return 42, nil })

	sched.ScheduleJob(ctx, shellJob, cronTrigger)
	sched.ScheduleJob(ctx, curlJob, quartz.NewSimpleTrigger(time.Second*7))
	sched.ScheduleJob(ctx, functionJob, quartz.NewSimpleTrigger(time.Second*3))

	time.Sleep(time.Second * 10)

	fmt.Println(sched.GetJobKeys())
	fmt.Println(shellJob.Result)

	responseBody, err := io.ReadAll(curlJob.Response.Body)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s\n%s\n", curlJob.Response.Status, string(responseBody))
	}
	fmt.Printf("Function job result: %v\n", *functionJob.Result)

	time.Sleep(time.Second * 2)
	sched.Stop()
	sched.Wait(ctx)
}
