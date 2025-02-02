# go-quartz

[![Build](https://github.com/reugn/go-quartz/actions/workflows/build.yml/badge.svg)](https://github.com/reugn/go-quartz/actions/workflows/build.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/reugn/go-quartz)](https://pkg.go.dev/github.com/reugn/go-quartz)
[![Go Report Card](https://goreportcard.com/badge/github.com/reugn/go-quartz)](https://goreportcard.com/report/github.com/reugn/go-quartz)
[![codecov](https://codecov.io/gh/reugn/go-quartz/branch/master/graph/badge.svg)](https://codecov.io/gh/reugn/go-quartz)

A minimalistic and zero-dependency scheduling library for Go.

## About

The implementation is inspired by the design of the [Quartz](https://github.com/quartz-scheduler/quartz)
Java scheduler.

### Library building blocks

#### Scheduler interface

```go
type Scheduler interface {
	// Start starts the scheduler. The scheduler will run until
	// the Stop method is called or the context is canceled. Use
	// the Wait method to block until all running jobs have completed.
	Start(context.Context)

	// IsStarted determines whether the scheduler has been started.
	IsStarted() bool

	// ScheduleJob schedules a job using the provided Trigger.
	ScheduleJob(jobDetail *JobDetail, trigger Trigger) error

	// GetJobKeys returns the keys of scheduled jobs.
	// For a job key to be returned, the job must satisfy all of the
	// matchers specified.
	// Given no matchers, it returns the keys of all scheduled jobs.
	GetJobKeys(...Matcher[ScheduledJob]) ([]*JobKey, error)

	// GetScheduledJob returns the scheduled job with the specified key.
	GetScheduledJob(jobKey *JobKey) (ScheduledJob, error)

	// DeleteJob removes the job with the specified key from the
	// scheduler's execution queue.
	DeleteJob(jobKey *JobKey) error

	// PauseJob suspends the job with the specified key from being
	// executed by the scheduler.
	PauseJob(jobKey *JobKey) error

	// ResumeJob restarts the suspended job with the specified key.
	ResumeJob(jobKey *JobKey) error

	// Clear removes all of the scheduled jobs.
	Clear() error

	// Wait blocks until the scheduler stops running and all jobs
	// have returned. Wait will return when the context passed to
	// it has expired. Until the context passed to start is
	// cancelled or Stop is called directly.
	Wait(context.Context)

	// Stop shutdowns the scheduler.
	Stop()
}
```

Implemented Schedulers

- StdScheduler

#### Trigger interface

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

#### Job interface

Any type that implements it can be scheduled.

```go
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated with this job fires.
	Execute(context.Context) error

	// Description returns the description of the Job.
	Description() string
}
```

Several common Job implementations can be found in the [job](./job) package.

## Cron expression format

| Field Name   | Mandatory | Allowed Values  | Allowed Special Characters |
|--------------|-----------|-----------------|----------------------------|
| Seconds      | YES       | 0-59            | , - * /                    |
| Minutes      | YES       | 0-59            | , - * /                    |
| Hours        | YES       | 0-23            | , - * /                    |
| Day of month | YES       | 1-31            | , - * ? / L W              |
| Month        | YES       | 1-12 or JAN-DEC | , - * /                    |
| Day of week  | YES       | 1-7 or SUN-SAT  | , - * ? / L #              |
| Year         | NO        | empty, 1970-    | , - * /                    |

## Distributed mode

The scheduler can use its own implementation of `quartz.JobQueue` to allow state sharing.  
An example implementation of the job queue using the file system as a persistence layer
can be found [here](./examples/queue/file_system.go).

## Examples

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create scheduler
	sched, _ := quartz.NewStdScheduler()

	// async start scheduler
	sched.Start(ctx)

	// create jobs
	cronTrigger, _ := quartz.NewCronTrigger("1/5 * * * * *")
	shellJob := job.NewShellJob("ls -la")

	request, _ := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	curlJob := job.NewCurlJob(request)

	functionJob := job.NewFunctionJob(func(_ context.Context) (int, error) { return 42, nil })

	// register jobs to scheduler
	_ = sched.ScheduleJob(quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob")),
		cronTrigger)
	_ = sched.ScheduleJob(quartz.NewJobDetail(curlJob, quartz.NewJobKey("curlJob")),
		quartz.NewSimpleTrigger(time.Second*7))
	_ = sched.ScheduleJob(quartz.NewJobDetail(functionJob, quartz.NewJobKey("functionJob")),
		quartz.NewSimpleTrigger(time.Second*5))

	// stop scheduler
	sched.Stop()

	// wait for all workers to exit
	sched.Wait(ctx)
}
```

More code samples can be found in the examples directory.

## License

Licensed under the MIT License.
