# go-quartz
[![Build](https://github.com/reugn/go-quartz/actions/workflows/build.yml/badge.svg)](https://github.com/reugn/go-quartz/actions/workflows/build.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/reugn/go-quartz)](https://pkg.go.dev/github.com/reugn/go-quartz)
[![Go Report Card](https://goreportcard.com/badge/github.com/reugn/go-quartz)](https://goreportcard.com/report/github.com/reugn/go-quartz)
[![codecov](https://codecov.io/gh/reugn/go-quartz/branch/master/graph/badge.svg)](https://codecov.io/gh/reugn/go-quartz)

A minimalistic and zero-dependency scheduling library for Go.

## About
Inspired by the [Quartz](https://github.com/quartz-scheduler/quartz) Java scheduler.

### Library building blocks
Scheduler interface
```go
type Scheduler interface {
	// Start starts the scheduler. The scheduler will run until
	// the Stop method is called or the context is canceled. Use
	// the Wait method to block until all running jobs have completed.
	Start(context.Context)
	// IsStarted determines whether the scheduler has been started.
	IsStarted() bool
	// ScheduleJob schedules a job using a specified trigger.
	ScheduleJob(job Job, trigger Trigger) error
	// GetJobKeys returns the keys of all of the scheduled jobs.
	GetJobKeys() []int
	// GetScheduledJob returns the scheduled job with the specified key.
	GetScheduledJob(key int) (*ScheduledJob, error)
	// DeleteJob removes the job with the specified key from the Scheduler's execution queue.
	DeleteJob(key int) error
	// Clear removes all of the scheduled jobs.
	Clear()
	// Stop shutdowns the scheduler.
	Stop()
	// Wait blocks until the scheduler stops running and all jobs
	// have returned. Wait will return when the context passed to
	// it has expired. Until the context passed to start is
	// cancelled or Stop is called directly.
	Wait(context.Context)
}
```
Implemented Schedulers
- StdScheduler

Trigger interface
```go
type Trigger interface {
	// NextFireTime returns the next time at which the Trigger is scheduled to fire.
	NextFireTime(prev int64) (int64, error)
	// Description returns the description of the Trigger.
	Description() string
}
```
Implemented Triggers
- CronTrigger
- SimpleTrigger
- RunOnceTrigger

Job interface. Any type that implements it can be scheduled.
```go
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated with this job fires.
	Execute(context.Context)
	// Description returns the description of the Job.
	Description() string
	// Key returns the unique key for the Job.
	Key() int
}
```
Implemented Jobs
- ShellJob
- CurlJob
- FunctionJob

## Cron expression format
| Field Name   | Mandatory | Allowed Values  | Allowed Special Characters |
| ------------ | --------- | --------------- | -------------------------- |
| Seconds      | YES       | 0-59            | , - * /                    |
| Minutes      | YES       | 0-59            | , - * /                    |
| Hours        | YES       | 0-23            | , - * /                    |
| Day of month | YES       | 1-31            | , - * ? /                  |
| Month        | YES       | 1-12 or JAN-DEC | , - * /                    |
| Day of week  | YES       | 1-7 or SUN-SAT  | , - * ? /                  |
| Year         | NO        | empty           | *                          |

## Examples
```go
ctx := context.Background()
sched := quartz.NewStdScheduler()
sched.Start(ctx)
cronTrigger, _ := quartz.NewCronTrigger("1/5 * * * * *")
shellJob := quartz.NewShellJob("ls -la")
curlJob, _ := quartz.NewCurlJob(http.MethodGet, "http://worldclockapi.com/api/json/est/now", "", nil)
functionJob := quartz.NewFunctionJob(func(_ context.Context) (int, error) { return 42, nil })
sched.ScheduleJob(shellJob, cronTrigger)
sched.ScheduleJob(curlJob, quartz.NewSimpleTrigger(time.Second*7))
sched.ScheduleJob(functionJob, quartz.NewSimpleTrigger(time.Second*5))
sched.Stop()
sched.Wait(ctx)
```
More code samples can be found in the examples directory.

## License
Licensed under the MIT License.
