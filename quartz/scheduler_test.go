package quartz_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestScheduler(t *testing.T) {
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

	sched.Start()
	sched.ScheduleJob(shellJob, quartz.NewSimpleTrigger(time.Millisecond*800))
	sched.ScheduleJob(curlJob, quartz.NewRunOnceTrigger(time.Millisecond))
	sched.ScheduleJob(errShellJob, quartz.NewRunOnceTrigger(time.Millisecond))
	sched.ScheduleJob(errCurlJob, quartz.NewSimpleTrigger(time.Millisecond*800))

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
	for _, tt := range []string{"Blocking", "NonBlocking"} {
		t.Run(tt, func(t *testing.T) {
			var opts quartz.StdSchedulerOptions
			switch tt {
			case "Blocking":
				opts.BlockingExecution = true
			case "NonBlocking":
				opts.BlockingExecution = false
			default:
				t.Fatal("unknown test", tt)
			}

			sched := quartz.NewStdSchedulerWithOptions(opts)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			sched.Start()

			var n int
			sched.ScheduleJob(quartz.NewFunctionJob(func() (bool, error) {
				n++
				timer := time.NewTimer(time.Hour)
				defer timer.Stop()
				select {
				case <-timer.C:
					return false, nil
				case <-ctx.Done():
					return true, nil
				}
			}), quartz.NewSimpleTrigger(time.Millisecond))

			ticker := time.NewTicker(2 * time.Millisecond)
			<-ticker.C
			if n == 0 {
				t.Error("job should have run once")
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
						if n != 1 {
							t.Error("job should have only run once", n)
						}
					}
				}
			case "NonBlocking":
				lastN := 0
			NONBLOCKING:
				for iters := 0; iters < 100; iters++ {
					select {
					case <-ctx.Done():
						break NONBLOCKING
					case <-ticker.C:
						if n <= lastN {
							t.Errorf("on iter %d n did not increase %d",
								iters, n,
							)
						}
						lastN = n
						t.Log(n)
					}
				}
			default:
				t.Fatal("unknown test:", tt)
			}

		})
	}
}
