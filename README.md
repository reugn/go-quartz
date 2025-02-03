# go-quartz

[![Build](https://github.com/reugn/go-quartz/actions/workflows/build.yml/badge.svg)](https://github.com/reugn/go-quartz/actions/workflows/build.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/reugn/go-quartz)](https://pkg.go.dev/github.com/reugn/go-quartz)
[![Go Report Card](https://goreportcard.com/badge/github.com/reugn/go-quartz)](https://goreportcard.com/report/github.com/reugn/go-quartz)
[![codecov](https://codecov.io/gh/reugn/go-quartz/branch/master/graph/badge.svg)](https://codecov.io/gh/reugn/go-quartz)

A minimalistic and zero-dependency scheduling library for Go.

## About

The implementation is inspired by the design of the [Quartz](https://github.com/quartz-scheduler/quartz)
Java scheduler.

The core [scheduler](#scheduler-interface) component can be used to manage scheduled [jobs](#job-interface) (tasks)
using [triggers](#trigger-interface).
The implementation of the cron trigger fully supports the Quartz [cron expression format](#cron-expression-format)
and can be used independently to calculate a future time given the previous execution time.

If you need to run multiple instances of the scheduler, see the [distributed mode](#distributed-mode) section for
guidance.

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

### Special characters

- `*`: All values in a field (e.g., `*` in minutes = "every minute").
- `?`: No specific value; use when specifying one of two related fields (e.g., "10" in day-of- month, `?` in
  day-of-week).
- `-`: Range of values (e.g., `10-12` in hour = "hours 10, 11, and 12").
- `,`: List of values (e.g., `MON,WED,FRI` in day-of-week = "Monday, Wednesday, Friday").
- `/`: Increments (e.g., `0/15` in seconds = "0, 15, 30, 45"; `1/3` in day-of-month = "every 3 days from the 1st").
- `L`: Last day; meaning varies by field. Ranges or lists are not allowed with `L`.
  - Day-of-month: Last day of the month (e.g, `L-3` is the third to last day of the month).
  - Day-of-week: Last day of the week (7 or SAT) when alone; "last xxx day" when used after
    another value (e.g., `6L` = "last Friday").
- `W`: Nearest weekday in the month to the given day (e.g., `15W` = "nearest weekday to the 15th"). If `1W` on
  Saturday, it fires Monday the 3rd. `W` only applies to a single day, not ranges or lists.
- `#`: Nth weekday of the month (e.g., `6#3` = "third Friday"; `2#1` = "first Monday"). Firing does not occur if
  that nth weekday does not exist in the month.

<sup>1</sup> The `L` and `W` characters can also be combined in the day-of-month field to yield `LW`, which
translates to "last weekday of the month".

<sup>2</sup> The names of months and days of the week are not case-sensitive. MON is the same as mon.

## Distributed mode

The scheduler can use its own implementation of `quartz.JobQueue` to allow state sharing.  
An example implementation of the job queue using the file system as a persistence layer
can be found [here](./examples/queue/file_system.go).

## Usage example

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a scheduler using the logger configuration option
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler, _ := quartz.NewStdScheduler(quartz.WithLogger(logger.NewSlogLogger(ctx, slogLogger)))

	// start the scheduler
	scheduler.Start(ctx)

	// create jobs
	cronTrigger, _ := quartz.NewCronTrigger("1/5 * * * * *")
	shellJob := job.NewShellJob("ls -la")

	request, _ := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	curlJob := job.NewCurlJob(request)

	functionJob := job.NewFunctionJob(func(_ context.Context) (int, error) { return 1, nil })

	// register the jobs with the scheduler
	_ = scheduler.ScheduleJob(quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob")),
		cronTrigger)
	_ = scheduler.ScheduleJob(quartz.NewJobDetail(curlJob, quartz.NewJobKey("curlJob")),
		quartz.NewSimpleTrigger(7*time.Second))
	_ = scheduler.ScheduleJob(quartz.NewJobDetail(functionJob, quartz.NewJobKey("functionJob")),
		quartz.NewSimpleTrigger(5*time.Second))

	// stop the scheduler
	scheduler.Stop()

	// wait for all workers to exit
	scheduler.Wait(ctx)
}
```

See the examples directory for additional code samples.

## License

Licensed under the MIT License.
