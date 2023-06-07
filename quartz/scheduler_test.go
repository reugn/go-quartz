package quartz_test

import (
	"context"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestScheduler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched := quartz.NewStdScheduler()
	var jobKeys [4]int

	shellJob := quartz.NewShellJob("ls -la")
	shellJob.Description()
	jobKeys[0] = shellJob.Key()

	curlJob, err := quartz.NewCurlJob(http.MethodGet, "http://worldclockapi.com/api/json/est/now", "", nil)
	assertEqual(t, err, nil)
	curlJob.Description()
	jobKeys[1] = curlJob.Key()

	errShellJob := quartz.NewShellJob("ls -z")
	jobKeys[2] = errShellJob.Key()

	errCurlJob, err := quartz.NewCurlJob(http.MethodGet, "http://", "", nil)
	assertEqual(t, err, nil)
	jobKeys[3] = errCurlJob.Key()

	sched.Start(ctx)
	sched.ScheduleJob(ctx, shellJob, quartz.NewSimpleTrigger(time.Millisecond*800))
	sched.ScheduleJob(ctx, curlJob, quartz.NewRunOnceTrigger(time.Millisecond))
	sched.ScheduleJob(ctx, errShellJob, quartz.NewRunOnceTrigger(time.Millisecond))
	sched.ScheduleJob(ctx, errCurlJob, quartz.NewSimpleTrigger(time.Millisecond*800))

	time.Sleep(time.Second)
	scheduledJobKeys := sched.GetJobKeys()
	assertEqual(t, scheduledJobKeys, []int{3668896347, 328790344})

	_, err = sched.GetScheduledJob(jobKeys[0])
	if err != nil {
		t.Fail()
	}

	err = sched.DeleteJob(shellJob.Key())
	if err != nil {
		t.Fail()
	}

	scheduledJobKeys = sched.GetJobKeys()
	assertEqual(t, len(scheduledJobKeys), 1)
	assertEqual(t, scheduledJobKeys, []int{328790344})

	sched.Clear()
	sched.Stop()
	assertEqual(t, shellJob.JobStatus, quartz.OK)
	// assertEqual(t, curlJob.JobStatus, quartz.OK)
	assertEqual(t, errShellJob.JobStatus, quartz.FAILURE)
	assertEqual(t, errCurlJob.JobStatus, quartz.FAILURE)
}

func TestSchedulerBlockingSemantics(t *testing.T) {
	for _, tt := range []string{"Blocking", "NonBlocking", "WorkerSmall", "WorkerLarge"} {
		t.Run(tt, func(t *testing.T) {
			var opts quartz.StdSchedulerOptions
			switch tt {
			case "Blocking":
				opts.BlockingExecution = true
			case "NonBlocking":
				opts.BlockingExecution = false
			case "WorkerSmall":
				opts.WorkerLimit = 4
			case "WorkerLarge":
				opts.WorkerLimit = 16
			default:
				t.Fatal("unknown semantic:", tt)
			}

			sched := quartz.NewStdSchedulerWithOptions(opts)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			sched.Start(ctx)

			var n int64
			sched.ScheduleJob(ctx,
				quartz.NewFunctionJob(func(ctx context.Context) (bool, error) {
					atomic.AddInt64(&n, 1)
					timer := time.NewTimer(time.Hour)
					defer timer.Stop()
					select {
					case <-timer.C:
						t.Error("should never reach this")
						return false, nil
					case <-ctx.Done():
						return true, nil
					}
				}),
				quartz.NewSimpleTrigger(time.Millisecond),
			)

			ticker := time.NewTicker(4 * time.Millisecond)
			defer ticker.Stop()
			<-ticker.C
			if atomic.LoadInt64(&n) == 0 {
				t.Error("job should have run at least once")
			}

			const attempts = 100
			switch tt {
			case "Blocking":
			BLOCKING:
				for iters := 0; iters < attempts; iters++ {
					iters++
					select {
					case <-ctx.Done():
						break BLOCKING
					case <-ticker.C:
						num := atomic.LoadInt64(&n)
						if num != 1 {
							t.Error("job should have only run once", num)
						}
					}
				}
			case "NonBlocking":
				var lastN int64
			NONBLOCKING:
				for iters := 0; iters < attempts; iters++ {
					select {
					case <-ctx.Done():
						break NONBLOCKING
					case <-ticker.C:
						num := atomic.LoadInt64(&n)
						if num > lastN {
							break NONBLOCKING
						}

						lastN = num
					}
				}
				num := atomic.LoadInt64(&n)
				if num <= lastN {
					t.Errorf("on iter %d n did not increase %d",
						attempts, num,
					)
				}

			case "WorkerSmall", "WorkerLarge":
			WORKERS:
				for iters := 0; iters < attempts; iters++ {
					select {
					case <-ctx.Done():
						break WORKERS
					case <-ticker.C:
						num := atomic.LoadInt64(&n)
						if num > int64(opts.WorkerLimit) {
							t.Errorf("on iter %d n %d was more than limit %d",
								iters, num, opts.WorkerLimit,
							)
						}
					}
				}
			default:
				t.Fatal("unknown test:", tt)
			}
		})
	}

}

func TestSchedulerCancel(t *testing.T) {
	hourJob := func(ctx context.Context) (bool, error) {
		timer := time.NewTimer(time.Hour)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
			return true, nil
		}
	}
	for _, tt := range []string{"context", "stop"} {
		// give the go runtime to exit many threads
		// before the second case.
		time.Sleep(time.Millisecond)
		t.Run("CloseMethod_"+tt, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			waitCtx, waitCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer waitCancel()

			startingRoutines := runtime.NumGoroutine()

			sched := quartz.NewStdScheduler()

			sched.Start(ctx)
			noopRoutines := runtime.NumGoroutine()
			if startingRoutines > noopRoutines {
				t.Error("should have started more threads",
					startingRoutines,
					noopRoutines,
				)
			}

			for i := 0; i < 100; i++ {
				if err := sched.ScheduleJob(ctx,
					quartz.NewFunctionJob(hourJob),
					quartz.NewSimpleTrigger(100*time.Millisecond),
				); err != nil {
					t.Errorf("could not add job %d, %s", i, err.Error())
				}
			}

			runningRoutines := runtime.NumGoroutine()
			if runningRoutines < noopRoutines {
				t.Error("number of running routines should not decrease",
					noopRoutines,
					runningRoutines,
				)
			}
			switch tt {
			case "context":
				cancel()
			case "stop":
				sched.Stop()
				time.Sleep(time.Millisecond) // trigger context switch
			default:
				t.Fatal("unknown test", tt)
			}

			// should not have timed out before we get to this point
			if err := waitCtx.Err(); err != nil {
				t.Fatal("test took too long")
			}

			sched.Wait(waitCtx)
			if err := waitCtx.Err(); err != nil {
				t.Fatal("waiting timed out before resources were released", err)
			}

			endingRoutines := runtime.NumGoroutine()
			if endingRoutines >= runningRoutines {
				t.Error("number of routines should decrease after wait",
					runningRoutines,
					endingRoutines,
				)
			}

			if t.Failed() {
				t.Log("starting", startingRoutines,
					"noop", noopRoutines,
					"running", runningRoutines,
					"ending", endingRoutines,
				)
			}
		})
	}
}
