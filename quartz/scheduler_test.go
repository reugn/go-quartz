package quartz_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/internal/mock"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func TestScheduler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched := quartz.NewStdScheduler()
	var jobKeys [4]*quartz.JobKey

	shellJob := job.NewShellJob("ls -la")
	jobKeys[0] = quartz.NewJobKey("shellJob")

	request, err := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	assert.Equal(t, err, nil)

	curlJob := job.NewCurlJobWithOptions(request, job.CurlJobOptions{HTTPClient: mock.HTTPHandlerOk})
	jobKeys[1] = quartz.NewJobKey("curlJob")

	errShellJob := job.NewShellJob("ls -z")
	jobKeys[2] = quartz.NewJobKey("errShellJob")

	request, err = http.NewRequest(http.MethodGet, "http://", nil)
	assert.Equal(t, err, nil)
	errCurlJob := job.NewCurlJob(request)
	jobKeys[3] = quartz.NewJobKey("errCurlJob")

	sched.Start(ctx)
	assert.Equal(t, sched.IsStarted(), true)

	err = sched.ScheduleJob(quartz.NewJobDetail(shellJob, jobKeys[0]),
		quartz.NewSimpleTrigger(time.Millisecond*700))
	assert.Equal(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(curlJob, jobKeys[1]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assert.Equal(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(errShellJob, jobKeys[2]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assert.Equal(t, err, nil)
	err = sched.ScheduleJob(quartz.NewJobDetail(errCurlJob, jobKeys[3]),
		quartz.NewSimpleTrigger(time.Millisecond*800))
	assert.Equal(t, err, nil)

	time.Sleep(time.Second)
	scheduledJobKeys := sched.GetJobKeys()
	assert.Equal(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[0], jobKeys[3]})

	_, err = sched.GetScheduledJob(jobKeys[0])
	assert.Equal(t, err, nil)

	err = sched.DeleteJob(jobKeys[0]) // shellJob key
	assert.Equal(t, err, nil)

	nonExistentJobKey := quartz.NewJobKey("NA")
	_, err = sched.GetScheduledJob(nonExistentJobKey)
	assert.NotEqual(t, err, nil)

	err = sched.DeleteJob(nonExistentJobKey)
	assert.NotEqual(t, err, nil)

	scheduledJobKeys = sched.GetJobKeys()
	assert.Equal(t, len(scheduledJobKeys), 1)
	assert.Equal(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[3]})

	_ = sched.Clear()
	assert.Equal(t, len(sched.GetJobKeys()), 0)
	sched.Stop()
	_, err = curlJob.DumpResponse(true)
	assert.Equal(t, err, nil)
	assert.Equal(t, shellJob.JobStatus(), job.StatusOK)
	assert.Equal(t, curlJob.JobStatus(), job.StatusOK)
	assert.Equal(t, errShellJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, errCurlJob.JobStatus(), job.StatusFailure)
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
				job.NewFunctionJob(func(ctx context.Context) (bool, error) {
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
				functionJob := quartz.NewJobDetail(job.NewFunctionJob(hourJob),
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

func TestSchedulerJobWithRetries(t *testing.T) {
	var n int32
	funcRetryJob := job.NewFunctionJob(func(_ context.Context) (string, error) {
		atomic.AddInt32(&n, 1)
		if n < 3 {
			return "", errors.New("less than 3")
		}
		return "ok", nil
	})
	ctx := context.Background()
	sched := quartz.NewStdScheduler()
	opts := quartz.NewDefaultJobDetailOptions()
	opts.MaxRetries = 3
	opts.RetryInterval = 50 * time.Millisecond
	jobDetail := quartz.NewJobDetailWithOptions(
		funcRetryJob,
		quartz.NewJobKey("funcRetryJob"),
		opts,
	)
	err := sched.ScheduleJob(jobDetail, quartz.NewRunOnceTrigger(time.Millisecond))
	assert.Equal(t, err, nil)

	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusNA)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 0)

	sched.Start(ctx)

	time.Sleep(25 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 1)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 2)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusOK)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 3)

	sched.Stop()
}

func TestSchedulerJobWithRetriesCtxDone(t *testing.T) {
	var n int32
	funcRetryJob := job.NewFunctionJob(func(_ context.Context) (string, error) {
		atomic.AddInt32(&n, 1)
		if n < 3 {
			return "", errors.New("less than 3")
		}
		return "ok", nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	sched := quartz.NewStdScheduler()
	opts := quartz.NewDefaultJobDetailOptions()
	opts.MaxRetries = 3
	opts.RetryInterval = 50 * time.Millisecond
	jobDetail := quartz.NewJobDetailWithOptions(
		funcRetryJob,
		quartz.NewJobKey("funcRetryJob"),
		opts,
	)
	err := sched.ScheduleJob(jobDetail, quartz.NewRunOnceTrigger(time.Millisecond))
	assert.Equal(t, err, nil)

	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusNA)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 0)

	sched.Start(ctx)

	time.Sleep(25 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 1)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 2)

	cancel() // cancel the context after first retry

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, funcRetryJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 2)

	sched.Stop()
}

func TestSchedulerArgumentValidationErrors(t *testing.T) {
	sched := quartz.NewStdScheduler()
	job := job.NewShellJob("ls -la")
	trigger := quartz.NewRunOnceTrigger(time.Millisecond)
	expiredTrigger, err := quartz.NewCronTrigger("0 0 0 1 1 ? 2023")
	assert.Equal(t, err, nil)

	err = sched.ScheduleJob(nil, trigger)
	assert.Equal(t, err.Error(), "jobDetail is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, nil), trigger)
	assert.Equal(t, err.Error(), "jobDetail.jobKey is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKey("")), trigger)
	assert.Equal(t, err.Error(), "empty key name is not allowed")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKeyWithGroup("job", "")), nil)
	assert.Equal(t, err.Error(), "trigger is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKey("job")), expiredTrigger)
	assert.Equal(t, err.Error(), "next trigger time is in the past")

	err = sched.DeleteJob(nil)
	assert.Equal(t, err.Error(), "jobKey is nil")

	_, err = sched.GetScheduledJob(nil)
	assert.Equal(t, err.Error(), "jobKey is nil")
}

func TestSchedulerStartStop(t *testing.T) {
	sched := quartz.NewStdScheduler()
	ctx := context.Background()
	sched.Start(ctx)
	sched.Start(ctx)
	assert.Equal(t, sched.IsStarted(), true)

	sched.Stop()
	sched.Stop()
	assert.Equal(t, sched.IsStarted(), false)
}
