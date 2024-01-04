package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	_ = sched.ScheduleJob(ctx, &PrintJob{"Ad hoc Job"}, quartz.NewRunOnceTrigger(time.Second*5))
	_ = sched.ScheduleJob(ctx, &PrintJob{"First job"}, quartz.NewSimpleTrigger(time.Second*12))
	_ = sched.ScheduleJob(ctx, &PrintJob{"Second job"}, quartz.NewSimpleTrigger(time.Second*6))
	_ = sched.ScheduleJob(ctx, &PrintJob{"Third job"}, quartz.NewSimpleTrigger(time.Second*3))
	_ = sched.ScheduleJob(ctx, &cronJob, cronTrigger)

	time.Sleep(time.Second * 10)

	scheduledJob, err := sched.GetScheduledJob(cronJob.Key())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(scheduledJob.Trigger().Description())
	fmt.Println("Before delete: ", sched.GetJobKeys())
	_ = sched.DeleteJob(ctx, cronJob.Key())
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
	curlJob := quartz.NewCurlJob(request)
	functionJob := quartz.NewFunctionJobWithDesc("42", func(_ context.Context) (int, error) { return 42, nil })

	_ = sched.ScheduleJob(ctx, shellJob, cronTrigger)
	_ = sched.ScheduleJob(ctx, curlJob, quartz.NewSimpleTrigger(time.Second*7))
	_ = sched.ScheduleJob(ctx, functionJob, quartz.NewSimpleTrigger(time.Second*3))

	time.Sleep(time.Second * 10)

	fmt.Println(sched.GetJobKeys())
	fmt.Println(shellJob.Stdout())

	response, err := curlJob.DumpResponse(true)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(response))
	}
	fmt.Printf("Function job result: %v\n", *functionJob.Result())

	time.Sleep(time.Second * 2)
	sched.Stop()
	sched.Wait(ctx)
}
