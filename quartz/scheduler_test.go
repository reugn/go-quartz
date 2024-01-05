package quartz_test

import (
	"context"
	"fmt"
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
	var jobKeys [4]*quartz.JobKey

	shellJob := quartz.NewShellJob("ls -la")
	shellJob.Description()
	jobKeys[0] = quartz.NewJobKey("shellJob")

	request, err := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	assertEqual(t, err, nil)

	handlerOk := struct{ httpHandlerMock }{}
	handlerOk.doFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Request:    request,
		}, nil
	}
	curlJob := quartz.NewCurlJobWithOptions(request, quartz.CurlJobOptions{HTTPClient: handlerOk})
	curlJob.Description()
	jobKeys[1] = quartz.NewJobKey("curlJob")

	errShellJob := quartz.NewShellJob("ls -z")
	jobKeys[2] = quartz.NewJobKey("errShellJob")

	request, err = http.NewRequest(http.MethodGet, "http://", nil)
	assertEqual(t, err, nil)
	errCurlJob := quartz.NewCurlJob(request)
	jobKeys[3] = quartz.NewJobKey("errCurlJob")

	sched.Start(ctx)
	assertEqual(t, sched.IsStarted(), true)

	err = sched.ScheduleJob(quartz.NewJobDetail(shellJob, jobKeys[0]),
		quartz.NewSimpleTrigger(time.Millisecond*700))
	assertEqual(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(curlJob, jobKeys[1]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assertEqual(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(errShellJob, jobKeys[2]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assertEqual(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(errCurlJob, jobKeys[3]),
		quartz.NewSimpleTrigger(time.Millisecond*800))
	assertEqual(t, err, nil)

	time.Sleep(time.Second)
	scheduledJobKeys := sched.GetJobKeys()
	assertEqual(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[0], jobKeys[3]})

	_, err = sched.GetScheduledJob(jobKeys[0])
	assertEqual(t, err, nil)

	err = sched.DeleteJob(jobKeys[0]) // shellJob key
	assertEqual(t, err, nil)

	nonExistentJobKey := quartz.NewJobKey("NA")
	_, err = sched.GetScheduledJob(nonExistentJobKey)
	assertNotEqual(t, err, nil)

	err = sched.DeleteJob(nonExistentJobKey)
	assertNotEqual(t, err, nil)

	scheduledJobKeys = sched.GetJobKeys()
	assertEqual(t, len(scheduledJobKeys), 1)
	assertEqual(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[3]})

	_ = sched.Clear()
	assertEqual(t, len(sched.GetJobKeys()), 0)
	sched.Stop()
	_, err = curlJob.DumpResponse(true)
	assertEqual(t, err, nil)
	assertEqual(t, shellJob.JobStatus(), quartz.OK)
	assertEqual(t, curlJob.JobStatus(), quartz.OK)
	assertEqual(t, errShellJob.JobStatus(), quartz.FAILURE)
	assertEqual(t, errCurlJob.JobStatus(), quartz.FAILURE)
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

			opts.OutdatedThreshold = 10 * time.Millisecond

			sched := quartz.NewStdSchedulerWithOptions(opts, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			sched.Start(ctx)

			var n int64
			timerJob := quartz.NewJobDetail(
				quartz.NewFunctionJob(func(ctx context.Context) (bool, error) {
					atomic.AddInt64(&n, 1)
					timer := time.NewTimer(time.Hour)
					defer timer.Stop()
					select {
					case <-timer.C:
						return false, nil
					case <-ctx.Done():
						return true, nil
					}
				}),
				quartz.NewJobKey("timerJob"),
			)
			err := sched.ScheduleJob(
				timerJob,
				quartz.NewSimpleTrigger(20*time.Millisecond),
			)
			if err != nil {
				t.Fatalf("Failed to schedule job, err: %s", err)
			}
			ticker := time.NewTicker(100 * time.Millisecond)
			<-ticker.C
			if atomic.LoadInt64(&n) == 0 {
				t.Error("job should have run at least once")
			}

			switch tt {
			case "Blocking":
			BLOCKING:
				for iters := 0; iters < 100; iters++ {
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
				for iters := 0; iters < 100; iters++ {
					select {
					case <-ctx.Done():
						break NONBLOCKING
					case <-ticker.C:
						num := atomic.LoadInt64(&n)
						if num <= lastN {
							t.Errorf("on iter %d n did not increase %d",
								iters, num,
							)
						}
						lastN = num
					}
				}
			case "WorkerSmall", "WorkerLarge":
			WORKERS:
				for iters := 0; iters < 100; iters++ {
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
				functionJob := quartz.NewJobDetail(quartz.NewFunctionJob(hourJob),
					quartz.NewJobKey(fmt.Sprintf("functionJob_%d", i)))
				if err := sched.ScheduleJob(
					functionJob,
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
