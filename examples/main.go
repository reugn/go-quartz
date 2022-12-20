package main

import (
	"context"
	"fmt"
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

	sched.ScheduleJob(&PrintJob{"Ad hoc Job"}, quartz.NewRunOnceTrigger(time.Second*5))
	sched.ScheduleJob(&PrintJob{"First job"}, quartz.NewSimpleTrigger(time.Second*12))
	sched.ScheduleJob(&PrintJob{"Second job"}, quartz.NewSimpleTrigger(time.Second*6))
	sched.ScheduleJob(&PrintJob{"Third job"}, quartz.NewSimpleTrigger(time.Second*3))
	sched.ScheduleJob(&cronJob, cronTrigger)

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
	curlJob, err := quartz.NewCurlJob(http.MethodGet, "http://worldclockapi.com/api/json/est/now", "", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	functionJob := quartz.NewFunctionJobWithDesc("42", func(_ context.Context) (int, error) { return 42, nil })

	sched.ScheduleJob(shellJob, cronTrigger)
	sched.ScheduleJob(curlJob, quartz.NewSimpleTrigger(time.Second*7))
	sched.ScheduleJob(functionJob, quartz.NewSimpleTrigger(time.Second*3))

	time.Sleep(time.Second * 10)

	fmt.Println(sched.GetJobKeys())
	fmt.Println(shellJob.Result)
	fmt.Println(curlJob.Response)
	fmt.Println(functionJob.Result)

	time.Sleep(time.Second * 2)
	sched.Stop()
	sched.Wait(ctx)
}
