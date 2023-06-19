package main

import (
	"context"
	"fmt"
	"github.com/reugn/go-quartz/quartz"
	"sync"
	"time"
)

type CustomLogger struct{}

func (c *CustomLogger) Log(msg string) {
	fmt.Printf("[CUSTOM LOGGER] %s", msg)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		sched := quartz.NewStdSchedulerWithOptions(quartz.StdSchedulerOptions{}, &CustomLogger{})
		sched.Start(ctx)
		sched.ScheduleJob(ctx, quartz.NewFunctionJob[int](func(ctx context.Context) (int, error) {
			return 12, nil
		}), quartz.NewRunOnceTrigger(time.Second))

		time.Sleep(time.Second * 2)
	}(ctx, wg)

	wg.Wait()
}
