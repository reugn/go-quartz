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
	"github.com/reugn/go-quartz/matcher"
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
	assert.IsNil(t, err)

	curlJob := job.NewCurlJobWithOptions(request, job.CurlJobOptions{HTTPClient: mock.HTTPHandlerOk})
	jobKeys[1] = quartz.NewJobKey("curlJob")

	errShellJob := job.NewShellJob("ls -z")
	jobKeys[2] = quartz.NewJobKey("errShellJob")

	request, err = http.NewRequest(http.MethodGet, "http://", nil)
	assert.IsNil(t, err)
	errCurlJob := job.NewCurlJob(request)
	jobKeys[3] = quartz.NewJobKey("errCurlJob")

	sched.Start(ctx)
	assert.Equal(t, sched.IsStarted(), true)

	err = sched.ScheduleJob(quartz.NewJobDetail(shellJob, jobKeys[0]),
		quartz.NewSimpleTrigger(time.Millisecond*700))
	assert.IsNil(t, err)
	err = sched.ScheduleJob(quartz.NewJobDetail(curlJob, jobKeys[1]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assert.IsNil(t, err)
	err = sched.ScheduleJob(quartz.NewJobDetail(errShellJob, jobKeys[2]),
		quartz.NewRunOnceTrigger(time.Millisecond))
	assert.IsNil(t, err)
	err = sched.ScheduleJob(quartz.NewJobDetail(errCurlJob, jobKeys[3]),
		quartz.NewSimpleTrigger(time.Millisecond*800))
	assert.IsNil(t, err)

	time.Sleep(time.Second)
	scheduledJobKeys, err := sched.GetJobKeys()
	assert.IsNil(t, err)
	assert.Equal(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[0], jobKeys[3]})

	_, err = sched.GetScheduledJob(jobKeys[0])
	assert.IsNil(t, err)

	err = sched.DeleteJob(jobKeys[0]) // shellJob key
	assert.IsNil(t, err)

	nonExistentJobKey := quartz.NewJobKey("NA")
	_, err = sched.GetScheduledJob(nonExistentJobKey)
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)

	err = sched.DeleteJob(nonExistentJobKey)
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)

	scheduledJobKeys, err = sched.GetJobKeys()
	assert.IsNil(t, err)
	assert.Equal(t, len(scheduledJobKeys), 1)
	assert.Equal(t, scheduledJobKeys, []*quartz.JobKey{jobKeys[3]})

	_ = sched.Clear()
	assert.Equal(t, jobCount(sched), 0)
	sched.Stop()
	_, err = curlJob.DumpResponse(true)
	assert.IsNil(t, err)
	assert.Equal(t, shellJob.JobStatus(), job.StatusOK)
	assert.Equal(t, curlJob.JobStatus(), job.StatusOK)
	assert.Equal(t, errShellJob.JobStatus(), job.StatusFailure)
	assert.Equal(t, errCurlJob.JobStatus(), job.StatusFailure)
}

func TestScheduler_BlockingSemantics(t *testing.T) {
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

func TestScheduler_Cancel(t *testing.T) {
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

			time.Sleep(5 * time.Millisecond)
			noopRoutines := runtime.NumGoroutine()
			if startingRoutines >= noopRoutines {
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

			time.Sleep(5 * time.Millisecond)
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

func TestScheduler_JobWithRetries(t *testing.T) {
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
	assert.IsNil(t, err)
	err = sched.ScheduleJob(jobDetail, quartz.NewRunOnceTrigger(time.Millisecond))
	assert.ErrorIs(t, err, quartz.ErrIllegalState)
	jobDetail.Options().Replace = true
	err = sched.ScheduleJob(jobDetail, quartz.NewRunOnceTrigger(time.Millisecond))
	assert.IsNil(t, err)

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

func TestScheduler_JobWithRetriesCtxDone(t *testing.T) {
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
	assert.IsNil(t, err)

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

func TestScheduler_MisfiredJob(t *testing.T) {
	funcJob := job.NewFunctionJob(func(_ context.Context) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return "ok", nil
	})

	misfiredChan := make(chan quartz.ScheduledJob, 1)
	sched := quartz.NewStdSchedulerWithOptions(quartz.StdSchedulerOptions{
		BlockingExecution: true,
		OutdatedThreshold: time.Millisecond,
		RetryInterval:     time.Millisecond,
		MisfiredChan:      misfiredChan,
	}, nil)

	jobDetail := quartz.NewJobDetail(funcJob, quartz.NewJobKey("funcJob"))
	err := sched.ScheduleJob(jobDetail, quartz.NewSimpleTrigger(2*time.Millisecond))
	assert.IsNil(t, err)

	sched.Start(context.Background())

	job := <-misfiredChan
	assert.Equal(t, job.JobDetail().JobKey().Name(), "funcJob")

	sched.Stop()
}

func TestScheduler_PauseResume(t *testing.T) {
	var n int32
	funcJob := job.NewFunctionJob(func(_ context.Context) (string, error) {
		atomic.AddInt32(&n, 1)
		return "ok", nil
	})
	sched := quartz.NewStdScheduler()
	jobDetail := quartz.NewJobDetail(funcJob, quartz.NewJobKey("funcJob"))
	err := sched.ScheduleJob(jobDetail, quartz.NewSimpleTrigger(10*time.Millisecond))
	assert.IsNil(t, err)

	assert.Equal(t, int(atomic.LoadInt32(&n)), 0)
	sched.Start(context.Background())

	time.Sleep(55 * time.Millisecond)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 5)

	err = sched.PauseJob(jobDetail.JobKey())
	assert.IsNil(t, err)

	time.Sleep(55 * time.Millisecond)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 5)

	err = sched.ResumeJob(jobDetail.JobKey())
	assert.IsNil(t, err)

	time.Sleep(55 * time.Millisecond)
	assert.Equal(t, int(atomic.LoadInt32(&n)), 10)

	sched.Stop()
}

func TestScheduler_PauseResumeErrors(t *testing.T) {
	funcJob := job.NewFunctionJob(func(_ context.Context) (string, error) {
		return "ok", nil
	})
	sched := quartz.NewStdScheduler()
	jobDetail := quartz.NewJobDetail(funcJob, quartz.NewJobKey("funcJob"))
	err := sched.ScheduleJob(jobDetail, quartz.NewSimpleTrigger(10*time.Millisecond))
	assert.IsNil(t, err)

	err = sched.ResumeJob(jobDetail.JobKey())
	assert.ErrorIs(t, err, quartz.ErrIllegalState)
	err = sched.ResumeJob(quartz.NewJobKey("funcJob2"))
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)

	err = sched.PauseJob(jobDetail.JobKey())
	assert.IsNil(t, err)
	err = sched.PauseJob(jobDetail.JobKey())
	assert.ErrorIs(t, err, quartz.ErrIllegalState)
	err = sched.PauseJob(quartz.NewJobKey("funcJob2"))
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)

	assert.Equal(t, jobCount(sched, matcher.JobPaused()), 1)
	assert.Equal(t, jobCount(sched, matcher.JobActive()), 0)
	assert.Equal(t, jobCount(sched), 1)

	sched.Stop()
}

func TestScheduler_ArgumentValidationErrors(t *testing.T) {
	sched := quartz.NewStdScheduler()
	job := job.NewShellJob("ls -la")
	trigger := quartz.NewRunOnceTrigger(time.Millisecond)
	expiredTrigger, err := quartz.NewCronTrigger("0 0 0 1 1 ? 2023")
	assert.IsNil(t, err)

	err = sched.ScheduleJob(nil, trigger)
	assert.ErrorContains(t, err, "jobDetail is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, nil), trigger)
	assert.ErrorContains(t, err, "jobDetail.jobKey is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKey("")), trigger)
	assert.ErrorContains(t, err, "empty key name is not allowed")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKeyWithGroup("job", "")), nil)
	assert.ErrorContains(t, err, "trigger is nil")
	err = sched.ScheduleJob(quartz.NewJobDetail(job, quartz.NewJobKey("job")), expiredTrigger)
	assert.ErrorIs(t, err, quartz.ErrTriggerExpired)

	err = sched.DeleteJob(nil)
	assert.ErrorContains(t, err, "jobKey is nil")

	err = sched.PauseJob(nil)
	assert.ErrorContains(t, err, "jobKey is nil")

	err = sched.ResumeJob(nil)
	assert.ErrorContains(t, err, "jobKey is nil")

	_, err = sched.GetScheduledJob(nil)
	assert.ErrorContains(t, err, "jobKey is nil")

	sched.Stop()
}

func TestScheduler_StartStop(t *testing.T) {
	sched := quartz.NewStdScheduler()
	ctx := context.Background()
	sched.Start(ctx)
	sched.Start(ctx)
	assert.Equal(t, sched.IsStarted(), true)

	sched.Stop()
	sched.Stop()
	assert.Equal(t, sched.IsStarted(), false)
}

func jobCount(sched quartz.Scheduler, matchers ...quartz.Matcher[quartz.ScheduledJob]) int {
	keys, _ := sched.GetJobKeys(matchers...)
	return len(keys)
}
