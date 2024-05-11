package quartz

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/reugn/go-quartz/logger"
)

// ScheduledJob represents a scheduled Job with the Trigger associated
// with it and the next run epoch time.
type ScheduledJob interface {
	JobDetail() *JobDetail
	Trigger() Trigger
	NextRunTime() int64
}

// Scheduler represents a Job orchestrator.
// Schedulers are responsible for executing Jobs when their associated
// Triggers fire (when their scheduled time arrives).
type Scheduler interface {
	// Start starts the scheduler. The scheduler will run until
	// the Stop method is called or the context is canceled. Use
	// the Wait method to block until all running jobs have completed.
	Start(context.Context)

	// IsStarted determines whether the scheduler has been started.
	IsStarted() bool

	// ScheduleJob schedules a job using a specified trigger.
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

// StdScheduler implements the quartz.Scheduler interface.
type StdScheduler struct {
	mtx       sync.Mutex
	wg        sync.WaitGroup
	queue     JobQueue
	queueMtx  sync.Locker
	interrupt chan struct{}
	cancel    context.CancelFunc
	feeder    chan ScheduledJob
	dispatch  chan ScheduledJob
	started   bool
	opts      StdSchedulerOptions
}

type StdSchedulerOptions struct {
	// When true, the scheduler will run jobs synchronously, waiting
	// for each execution instance of the job to return before starting
	// the next execution. Running with this option effectively serializes
	// all job execution.
	BlockingExecution bool

	// When greater than 0, all jobs will be dispatched to a pool of
	// goroutines of WorkerLimit size to limit the total number of processes
	// usable by the scheduler. If all worker threads are in use, job
	// scheduling will wait till a job can be dispatched.
	// If BlockingExecution is set, then WorkerLimit is ignored.
	WorkerLimit int

	// When the scheduler attempts to execute a job, if the time elapsed
	// since the job's scheduled execution time is less than or equal to the
	// configured threshold, the scheduler will execute the job.
	// Otherwise, the job will be rescheduled as outdated. By default,
	// NewStdScheduler sets the threshold to 100ms.
	//
	// As a rule of thumb, your OutdatedThreshold should always be
	// greater than 0, but less than the shortest interval used by
	// your job or jobs.
	OutdatedThreshold time.Duration

	// This retry interval will be used if the scheduler fails to
	// calculate the next time to interrupt for job execution. By default,
	// the NewStdScheduler constructor sets this interval to 100
	// milliseconds. Changing the default value may be beneficial when
	// using a custom implementation of the JobQueue, where operations
	// may timeout or fail.
	RetryInterval time.Duration

	// MisfiredChan allows the creation of event listeners to handle jobs that
	// have failed to be executed on time and have been skipped by the scheduler.
	//
	// Misfires can occur due to insufficient resources or scheduler downtime.
	// Adjust OutdatedThreshold to establish an acceptable delay time and
	// ensure regular job execution.
	MisfiredChan chan ScheduledJob
}

// Verify StdScheduler satisfies the Scheduler interface.
var _ Scheduler = (*StdScheduler)(nil)

// NewStdScheduler returns a new StdScheduler with the default configuration.
func NewStdScheduler() Scheduler {
	return NewStdSchedulerWithOptions(StdSchedulerOptions{
		OutdatedThreshold: 100 * time.Millisecond,
		RetryInterval:     100 * time.Millisecond,
	}, nil, nil)
}

// NewStdSchedulerWithOptions returns a new StdScheduler configured as specified.
//
// A custom [JobQueue] implementation may be provided to manage scheduled jobs.
// This is useful when distributed mode is required, so that jobs can be stored
// in persistent storage. Pass in nil to use the internal in-memory implementation.
//
// A custom [sync.Locker] may also be provided to ensure that scheduler operations
// on the job queue are atomic when used in distributed mode. Pass in nil to use
// the default [sync.Mutex].
func NewStdSchedulerWithOptions(
	opts StdSchedulerOptions,
	jobQueue JobQueue,
	jobQueueMtx sync.Locker,
) *StdScheduler {
	if jobQueue == nil {
		jobQueue = NewJobQueue()
	}
	if jobQueueMtx == nil {
		jobQueueMtx = &sync.Mutex{}
	}
	return &StdScheduler{
		queue:     jobQueue,
		queueMtx:  jobQueueMtx,
		interrupt: make(chan struct{}, 1),
		feeder:    make(chan ScheduledJob),
		dispatch:  make(chan ScheduledJob),
		opts:      opts,
	}
}

// ScheduleJob schedules a Job using a specified Trigger.
func (sched *StdScheduler) ScheduleJob(
	jobDetail *JobDetail,
	trigger Trigger,
) error {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	if jobDetail == nil {
		return illegalArgumentError("jobDetail is nil")
	}
	if jobDetail.jobKey == nil {
		return illegalArgumentError("jobDetail.jobKey is nil")
	}
	if jobDetail.jobKey.name == "" {
		return illegalArgumentError("empty key name is not allowed")
	}
	if trigger == nil {
		return illegalArgumentError("trigger is nil")
	}
	nextRunTime := int64(math.MaxInt64)
	var err error
	if !jobDetail.opts.Suspended {
		nextRunTime, err = trigger.NextFireTime(NowNano())
		if err != nil {
			return err
		}
	}
	toSchedule := &scheduledJob{
		job:      jobDetail,
		trigger:  trigger,
		priority: nextRunTime,
	}
	err = sched.queue.Push(toSchedule)
	if err == nil {
		logger.Debugf("Successfully added job %s.", jobDetail.jobKey)
		if sched.started {
			sched.Reset()
		}
	}
	return err
}

// Start starts the StdScheduler execution loop.
func (sched *StdScheduler) Start(ctx context.Context) {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	if sched.started {
		logger.Info("Scheduler is already running.")
		return
	}

	ctx, sched.cancel = context.WithCancel(ctx)
	go func() { <-ctx.Done(); sched.Stop() }()

	// start scheduler execution loop
	sched.wg.Add(1)
	go sched.startExecutionLoop(ctx)

	// starts worker pool when WorkerLimit is greater than 0
	sched.startWorkers(ctx)

	sched.started = true
}

// Wait blocks until the scheduler shuts down.
func (sched *StdScheduler) Wait(ctx context.Context) {
	sig := make(chan struct{})
	go func() { defer close(sig); sched.wg.Wait() }()
	select {
	case <-ctx.Done():
	case <-sig:
	}
}

// IsStarted determines whether the scheduler has been started.
func (sched *StdScheduler) IsStarted() bool {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	return sched.started
}

// GetJobKeys returns the keys of scheduled jobs.
// For a job key to be returned, the job must satisfy all of the matchers specified.
// Given no matchers, it returns the keys of all scheduled jobs.
func (sched *StdScheduler) GetJobKeys(matchers ...Matcher[ScheduledJob]) ([]*JobKey, error) {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	scheduledJobs, err := sched.queue.ScheduledJobs(matchers)
	if err != nil {
		return nil, err
	}
	keys := make([]*JobKey, 0, len(scheduledJobs))
	for _, scheduled := range scheduledJobs {
		keys = append(keys, scheduled.JobDetail().jobKey)
	}
	return keys, nil
}

// GetScheduledJob returns the ScheduledJob with the specified key.
func (sched *StdScheduler) GetScheduledJob(jobKey *JobKey) (ScheduledJob, error) {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	if jobKey == nil {
		return nil, illegalArgumentError("jobKey is nil")
	}
	return sched.queue.Get(jobKey)
}

// DeleteJob removes the Job with the specified key if present.
func (sched *StdScheduler) DeleteJob(jobKey *JobKey) error {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	if jobKey == nil {
		return illegalArgumentError("jobKey is nil")
	}
	_, err := sched.queue.Remove(jobKey)
	if err == nil {
		logger.Debugf("Successfully deleted job %s.", jobKey)
		if sched.started {
			sched.Reset()
		}
	}
	return err
}

// PauseJob suspends the job with the specified key from being
// executed by the scheduler.
func (sched *StdScheduler) PauseJob(jobKey *JobKey) error {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	if jobKey == nil {
		return illegalArgumentError("jobKey is nil")
	}
	job, err := sched.queue.Get(jobKey)
	if err != nil {
		return err
	}
	if job.JobDetail().opts.Suspended {
		return illegalStateError(fmt.Sprintf("job %s is suspended", jobKey))
	}
	job, err = sched.queue.Remove(jobKey)
	if err == nil {
		job.JobDetail().opts.Suspended = true
		paused := &scheduledJob{
			job:      job.JobDetail(),
			trigger:  job.Trigger(),
			priority: int64(math.MaxInt64),
		}
		err = sched.queue.Push(paused)
		if err == nil {
			logger.Debugf("Successfully paused job %s.", jobKey)
			if sched.started {
				sched.Reset()
			}
		}
	}
	return err
}

// ResumeJob restarts the suspended job with the specified key.
func (sched *StdScheduler) ResumeJob(jobKey *JobKey) error {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	if jobKey == nil {
		return illegalArgumentError("jobKey is nil")
	}
	job, err := sched.queue.Get(jobKey)
	if err != nil {
		return err
	}
	if !job.JobDetail().opts.Suspended {
		return illegalStateError(fmt.Sprintf("job %s is active", jobKey))
	}
	job, err = sched.queue.Remove(jobKey)
	if err == nil {
		job.JobDetail().opts.Suspended = false
		nextRunTime, err := job.Trigger().NextFireTime(NowNano())
		if err != nil {
			return err
		}
		resumed := &scheduledJob{
			job:      job.JobDetail(),
			trigger:  job.Trigger(),
			priority: nextRunTime,
		}
		err = sched.queue.Push(resumed)
		if err == nil {
			logger.Debugf("Successfully resumed job %s.", jobKey)
			if sched.started {
				sched.Reset()
			}
		}
	}
	return err
}

// Clear removes all of the scheduled jobs.
func (sched *StdScheduler) Clear() error {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	// reset the job queue
	err := sched.queue.Clear()
	if err == nil {
		logger.Debug("Successfully cleared job queue.")
		if sched.started {
			sched.Reset()
		}
	}
	return err
}

// Stop exits the StdScheduler execution loop.
func (sched *StdScheduler) Stop() {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	if !sched.started {
		logger.Info("Scheduler is not running.")
		return
	}

	logger.Info("Closing the StdScheduler.")
	sched.cancel()
	sched.started = false
}

func (sched *StdScheduler) startExecutionLoop(ctx context.Context) {
	defer sched.wg.Done()
	maxTimerDuration := time.Duration(1<<63 - 1)
	timer := time.NewTimer(maxTimerDuration)
	for {
		queueSize, err := sched.queue.Size()
		switch {
		case err != nil:
			logger.Errorf("Failed to fetch queue size: %s", err)
			timer.Reset(sched.opts.RetryInterval)
		case queueSize == 0:
			logger.Trace("Queue is empty.")
			timer.Reset(maxTimerDuration)
		default:
			timer.Reset(sched.calculateNextTick())
		}
		select {
		case <-timer.C:
			logger.Trace("Tick.")
			sched.executeAndReschedule(ctx)

		case <-sched.interrupt:
			logger.Trace("Interrupted waiting for next tick.")
			timer.Stop()

		case <-ctx.Done():
			logger.Info("Exit the execution loop.")
			timer.Stop()
			return
		}
	}
}

func (sched *StdScheduler) startWorkers(ctx context.Context) {
	if sched.opts.WorkerLimit > 0 {
		logger.Debugf("Starting %d scheduler workers.", sched.opts.WorkerLimit)
		for i := 0; i < sched.opts.WorkerLimit; i++ {
			sched.wg.Add(1)
			go func() {
				defer sched.wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case scheduled := <-sched.dispatch:
						executeWithRetries(ctx, scheduled.JobDetail())
					}
				}
			}()
		}
	}
}

func (sched *StdScheduler) calculateNextTick() time.Duration {
	var nextTickDuration time.Duration
	scheduledJob, err := sched.queue.Head()
	if err != nil {
		if errors.Is(err, ErrQueueEmpty) {
			logger.Debug("Queue is empty")
			return nextTickDuration
		}
		logger.Errorf("Failed to calculate next tick: %s", err)
		return sched.opts.RetryInterval
	}
	nextRunTime := scheduledJob.NextRunTime()
	now := NowNano()
	if nextRunTime > now {
		nextTickDuration = time.Duration(nextRunTime - now)
	}
	logger.Tracef("Next tick is for %s in %s.", scheduledJob.JobDetail().jobKey,
		nextTickDuration)

	return nextTickDuration
}

func (sched *StdScheduler) executeAndReschedule(ctx context.Context) {
	// fetch a job for processing
	scheduled, valid := sched.fetchAndReschedule()

	// execute the job
	if valid {
		logger.Debugf("Job %s is about to be executed.", scheduled.JobDetail().jobKey)
		switch {
		case sched.opts.BlockingExecution:
			executeWithRetries(ctx, scheduled.JobDetail())
		case sched.opts.WorkerLimit > 0:
			select {
			case sched.dispatch <- scheduled:
			case <-ctx.Done():
				return
			}
		default:
			sched.wg.Add(1)
			go func() {
				defer sched.wg.Done()
				executeWithRetries(ctx, scheduled.JobDetail())
			}()
		}
	}
}

func executeWithRetries(ctx context.Context, jobDetail *JobDetail) {
	// recover from unhandled panics that may occur during job execution
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("Job %s panicked: %s", jobDetail.jobKey, err)
		}
	}()

	err := jobDetail.job.Execute(ctx)
	if err == nil {
		return
	}
retryLoop:
	for i := 1; i <= jobDetail.opts.MaxRetries; i++ {
		timer := time.NewTimer(jobDetail.opts.RetryInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			break retryLoop
		}
		logger.Tracef("Job %s retry %d", jobDetail.jobKey, i)
		err = jobDetail.job.Execute(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		logger.Warnf("Job %s terminated with error: %s", jobDetail.jobKey, err)
	}
}

func (sched *StdScheduler) validateJob(job ScheduledJob) (bool, func() (int64, error)) {
	if job.JobDetail().opts.Suspended {
		return false, func() (int64, error) { return math.MaxInt64, nil }
	}
	now := NowNano()
	if job.NextRunTime() < now-sched.opts.OutdatedThreshold.Nanoseconds() {
		duration := time.Duration(now - job.NextRunTime())
		logger.Infof("Job %s is outdated %s.", job.JobDetail().jobKey, duration)
		select {
		case sched.opts.MisfiredChan <- job:
		default:
		}
		return false, func() (int64, error) { return job.Trigger().NextFireTime(now) }
	} else if job.NextRunTime() > now {
		logger.Debugf("Job %s is not due to run yet.", job.JobDetail().jobKey)
		return false, func() (int64, error) { return job.NextRunTime(), nil }
	}
	return true, func() (int64, error) { return job.Trigger().NextFireTime(job.NextRunTime()) }
}

func (sched *StdScheduler) fetchAndReschedule() (ScheduledJob, bool) {
	sched.queueMtx.Lock()
	defer sched.queueMtx.Unlock()

	// fetch a job for processing
	job, err := sched.queue.Pop()
	if err != nil {
		if errors.Is(err, ErrQueueEmpty) {
			logger.Debug("Queue is empty")
		} else {
			logger.Errorf("Failed to fetch a job from the queue: %s", err)
		}
		return nil, false
	}
	// validate the job
	valid, nextRunTimeExtractor := sched.validateJob(job)
	// calculate next run time for the job
	nextRunTime, err := nextRunTimeExtractor()
	if err != nil {
		logger.Infof("Job %s exited the execution loop: %s.", job.JobDetail().jobKey, err)
		return job, valid
	}
	// reschedule the job
	toSchedule := &scheduledJob{
		job:      job.JobDetail(),
		trigger:  job.Trigger(),
		priority: nextRunTime,
	}
	if err := sched.queue.Push(toSchedule); err != nil {
		logger.Errorf("Failed to reschedule job %s, err: %s",
			toSchedule.JobDetail().jobKey, err)
	} else {
		logger.Tracef("Successfully rescheduled job %s", toSchedule.JobDetail().jobKey)
		sched.Reset()
	}

	return job, valid
}

// Reset is called internally to recalculate the closest job timing when there
// is an update to the job queue by the scheduler. In cluster mode with a shared
// queue, it can be triggered manually to synchronize with remote changes if one
// of the schedulers fails.
func (sched *StdScheduler) Reset() {
	select {
	case sched.interrupt <- struct{}{}:
	default:
	}
}
