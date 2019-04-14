# go-quartz
A go scheduling library

## About
Inspired by [quartz](https://github.com/quartz-scheduler/quartz) java scheduler.

### Library building blocks
Job interface. Should be implemented by custom jobs for further scheduling
```go
type Job interface {
	Execute()
	Description() string
}
```
Scheduler interface
```go
type Scheduler interface {
	Start()
	ScheduleJob(job Job, trigger Trigger) error
	Stop()
}
```
Implemented Schedulers
- StdScheduler

Trigger interface
```go
type Trigger interface {
	NextFireTime(prev int64) (int64, error)
	Description() string
}
```
Implemented Triggers
- CronTrigger
- SimpleTrigger
- RunOnceTrigger

## Examples
Could be found in examples directory.

## License
Licensed under the MIT License.
