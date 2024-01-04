package quartz

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/reugn/go-quartz/quartz/logger"
)

// ScheduledJob represents a scheduled Job with the Trigger associated
// with it and the next run epoch time.
type ScheduledJob interface {
	Job() Job
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
	ScheduleJob(ctx context.Context, job Job, trigger Trigger) error

	// GetJobKeys returns the keys of all of the scheduled jobs.
	GetJobKeys() []int

	// GetScheduledJob returns the scheduled job with the specified key.
	GetScheduledJob(key int) (ScheduledJob, error)

	// DeleteJob removes the job with the specified key from the
	// scheduler's execution queue.
	DeleteJob(ctx context.Context, key int) error

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
	interrupt chan struct{}
	cancel    context.CancelFunc
	feeder    chan ScheduledJob
	dispatch  chan ScheduledJob
	started   bool
	opts      StdSchedulerOptions
}

type StdSchedulerOptions struct {
	// When true, the scheduler will run jobs synchronously,
	// waiting for each execution instance of the job to return
	// before starting the next execution. Running with this
	// option effectively serializes all job execution.
	BlockingExecution bool

	// When greater than 0, all jobs will be dispatched to a pool
	// of goroutines of WorkerLimit size to limit the total number
	// of processes usable by the Scheduler. If all worker threads
	// are in use, job scheduling will wait till a job can be
	// dispatched. If BlockingExecution is set, then WorkerLimit
	// is ignored.
	WorkerLimit int

	// When the scheduler attempts to schedule a job, if the job
	// is due to run in less than or equal to this value, then the
	// scheduler will run the job, even if the "next scheduled
	// job" is in the future. Historically, Go-Quartz had a
	// scheduled time of 30 seconds, by default (NewStdScheduler)
	// has a threshold of 100ms (if a job will be "triggered" in
	// 100ms, then it is run now.)
	//
	// As a rule of thumb, your OutdatedThreshold should always be
	// greater than 0, but less than the shortest interval used by
	// your job or jobs.
	OutdatedThreshold time.Duration
}

// Verify StdScheduler satisfies the Scheduler interface.
var _ Scheduler = (*StdScheduler)(nil)

// NewStdScheduler returns a new StdScheduler with the default
// configuration.
func NewStdScheduler() Scheduler {
	return NewStdSchedulerWithOptions(StdSchedulerOptions{
		OutdatedThreshold: 100 * time.Millisecond,
	}, nil)
}

// NewStdSchedulerWithOptions returns a new StdScheduler configured
// as specified.
// A custom JobQueue implementation can be provided to manage scheduled
// jobs. This can be useful when distributed mode is required, so that
// jobs can be stored in persistent storage.
// Pass in nil to use the internal in-memory implementation.
func NewStdSchedulerWithOptions(
	opts StdSchedulerOptions,
	jobQueue JobQueue,
) *StdScheduler {
	if jobQueue == nil {
		jobQueue = newJobQueue()
	}
	return &StdScheduler{
		queue:     jobQueue,
		interrupt: make(chan struct{}, 1),
		feeder:    make(chan ScheduledJob),
		dispatch:  make(chan ScheduledJob),
		opts:      opts,
	}
}

// ScheduleJob schedules a Job using a specified Trigger.
func (sched *StdScheduler) ScheduleJob(
	ctx context.Context,
	job Job,
	trigger Trigger,
) error {
	nextRunTime, err := trigger.NextFireTime(NowNano())
	if err != nil {
		return err
	}

	select {
	case sched.feeder <- &scheduledJob{
		job:      job,
		trigger:  trigger,
		priority: nextRunTime,
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
	// start the feed reader
	sched.wg.Add(1)
	go sched.startFeedReader(ctx)

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

// GetJobKeys returns the keys of all of the scheduled jobs.
func (sched *StdScheduler) GetJobKeys() []int {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	keys := make([]int, 0, sched.queue.Size())
	for _, scheduled := range sched.queue.ScheduledJobs() {
		keys = append(keys, scheduled.Job().Key())
	}

	return keys
}

// GetScheduledJob returns the ScheduledJob with the specified key.
func (sched *StdScheduler) GetScheduledJob(key int) (ScheduledJob, error) {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	for _, scheduled := range sched.queue.ScheduledJobs() {
		if scheduled.Job().Key() == key {
			return scheduled, nil
		}
	}

	return nil, errors.New("no Job with the given Key found")
}

// DeleteJob removes the Job with the specified key if present.
func (sched *StdScheduler) DeleteJob(ctx context.Context, key int) error {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	for i, scheduled := range sched.queue.ScheduledJobs() {
		if scheduled.Job().Key() == key {
			_, err := sched.queue.Remove(i)
			if err == nil {
				sched.reset(ctx)
			}
			return err
		}
	}

	return errors.New("no Job with the given Key found")
}

// Clear removes all of the scheduled jobs.
func (sched *StdScheduler) Clear() error {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	// reset the job queue
	return sched.queue.Clear()
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
	for {
		if sched.queueLen() == 0 {
			select {
			case <-sched.interrupt:
			case <-ctx.Done():
				logger.Info("Exit the empty execution loop.")
				return
			}
		} else {
			t := time.NewTimer(sched.calculateNextTick())
			select {
			case <-t.C:
				sched.executeAndReschedule(ctx)

			case <-sched.interrupt:
				t.Stop()

			case <-ctx.Done():
				logger.Info("Exit the execution loop.")
				t.Stop()
				return
			}
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
						scheduled.Job().Execute(ctx)
					}
				}
			}()
		}
	}
}

func (sched *StdScheduler) queueLen() int {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	return sched.queue.Size()
}

func (sched *StdScheduler) calculateNextTick() time.Duration {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	if sched.queue.Size() > 0 {
		scheduledJob, err := sched.queue.Head()
		if err != nil {
			logger.Warnf("Failed to calculate next tick: %s", err)
		} else {
			return time.Duration(parkTime(scheduledJob.NextRunTime()))
		}
	}
	return sched.opts.OutdatedThreshold
}

func (sched *StdScheduler) executeAndReschedule(ctx context.Context) {
	// return if the job queue is empty
	if sched.queueLen() == 0 {
		logger.Debug("Job queue is empty.")
		return
	}

	// fetch a job for processing
	sched.mtx.Lock()
	scheduled, err := sched.queue.Pop()
	if err != nil {
		logger.Errorf("Failed to fetch a job from the queue: %s", err)
		sched.mtx.Unlock()
		return
	}
	sched.mtx.Unlock()

	// check if the job is due to be processed
	if scheduled.NextRunTime() > NowNano() {
		logger.Tracef("Job %d is not due to run yet.", scheduled.Job().Key())
		return
	}

	// execute the job
	if sched.jobIsUpToDate(scheduled) {
		logger.Debugf("Job %d is about to be executed.", scheduled.Job().Key())
		switch {
		case sched.opts.BlockingExecution:
			scheduled.Job().Execute(ctx)
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
				scheduled.Job().Execute(ctx)
			}()
		}
	} else {
		logger.Debugf("Job %d skipped as outdated %d.", scheduled.Job().Key(),
			scheduled.NextRunTime())
	}

	// reschedule the job
	sched.rescheduleJob(ctx, scheduled)
}

func (sched *StdScheduler) rescheduleJob(ctx context.Context, job ScheduledJob) {
	nextRunTime, err := job.Trigger().NextFireTime(job.NextRunTime())
	if err != nil {
		logger.Infof("Job %d got out the execution loop: %s.", job.Job().Key(), err)
		return
	}

	select {
	case <-ctx.Done():
	case sched.feeder <- &scheduledJob{
		job:      job.Job(),
		trigger:  job.Trigger(),
		priority: nextRunTime,
	}:
	}
}

func (sched *StdScheduler) jobIsUpToDate(job ScheduledJob) bool {
	return job.NextRunTime() > NowNano()-sched.opts.OutdatedThreshold.Nanoseconds()
}

func (sched *StdScheduler) startFeedReader(ctx context.Context) {
	defer sched.wg.Done()
	for {
		select {
		case scheduled := <-sched.feeder:
			func() {
				sched.mtx.Lock()
				defer sched.mtx.Unlock()

				if err := sched.queue.Push(scheduled); err != nil {
					logger.Errorf("Failed to schedule job %d.", scheduled.Job().Key())
				} else {
					sched.reset(ctx)
				}
			}()
		case <-ctx.Done():
			logger.Info("Exit the feed reader.")
			return
		}
	}
}

func (sched *StdScheduler) reset(ctx context.Context) {
	select {
	case sched.interrupt <- struct{}{}:
	case <-ctx.Done():
	default:
	}
}

func parkTime(ts int64) int64 {
	now := NowNano()
	if ts > now {
		return ts - now
	}
	return 0
}
