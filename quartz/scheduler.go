package quartz

import (
	"context"
	"errors"
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

	// ScheduleJob schedules a job using the provided trigger.
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

// StdScheduler implements the [Scheduler] interface.
type StdScheduler struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	interrupt chan struct{}
	cancel    context.CancelFunc
	feeder    chan ScheduledJob
	dispatch  chan ScheduledJob
	started   bool

	queue       JobQueue
	queueLocker sync.Locker

	opts   schedulerConfig
	logger logger.Logger
}

var _ Scheduler = (*StdScheduler)(nil)

type schedulerConfig struct {
	// When true, the scheduler will run jobs synchronously, waiting
	// for each execution instance of the job to return before starting
	// the next execution. Running with this option effectively serializes
	// all job execution.
	blockingExecution bool

	// When greater than 0, all jobs will be dispatched to a pool of
	// goroutines of WorkerLimit size to limit the total number of processes
	// usable by the scheduler. If all worker threads are in use, job
	// scheduling will wait till a job can be dispatched.
	// If BlockingExecution is set, then WorkerLimit is ignored.
	workerLimit int

	// When the scheduler attempts to execute a job, if the time elapsed
	// since the job's scheduled execution time is less than or equal to the
	// configured threshold, the scheduler will execute the job.
	// Otherwise, the job will be rescheduled as outdated. By default,
	// NewStdScheduler sets the threshold to 100ms.
	//
	// As a rule of thumb, your OutdatedThreshold should always be
	// greater than 0, but less than the shortest interval used by
	// your job or jobs.
	outdatedThreshold time.Duration

	// This retry interval will be used if the scheduler fails to
	// calculate the next time to interrupt for job execution. By default,
	// the NewStdScheduler constructor sets this interval to 100
	// milliseconds. Changing the default value may be beneficial when
	// using a custom implementation of the JobQueue, where operations
	// may timeout or fail.
	retryInterval time.Duration

	// MisfiredChan allows the creation of event listeners to handle jobs that
	// have failed to be executed on time and have been skipped by the scheduler.
	//
	// Misfires can occur due to insufficient resources or scheduler downtime.
	// Adjust OutdatedThreshold to establish an acceptable delay time and
	// ensure regular job execution.
	misfiredChan chan ScheduledJob
}

// SchedulerOpt is a functional option type used to configure an [StdScheduler].
type SchedulerOpt func(*StdScheduler) error

// WithBlockingExecution configures the scheduler to use blocking execution.
// In blocking execution mode, jobs are executed synchronously in the scheduler's
// main loop.
func WithBlockingExecution() SchedulerOpt {
	return func(c *StdScheduler) error {
		c.opts.blockingExecution = true
		return nil
	}
}

// WithWorkerLimit configures the number of worker goroutines for concurrent job execution.
// This option is only used when blocking execution is disabled. If blocking execution
// is enabled, this setting will be ignored. The workerLimit must be non-negative.
func WithWorkerLimit(workerLimit int) SchedulerOpt {
	return func(c *StdScheduler) error {
		if workerLimit < 0 {
			return newIllegalArgumentError("workerLimit must be non-negative")
		}
		c.opts.workerLimit = workerLimit
		return nil
	}
}

// WithOutdatedThreshold configures the time duration after which a scheduled job is
// considered outdated.
func WithOutdatedThreshold(outdatedThreshold time.Duration) SchedulerOpt {
	return func(c *StdScheduler) error {
		c.opts.outdatedThreshold = outdatedThreshold
		return nil
	}
}

// WithRetryInterval configures the time interval the scheduler waits before
// retrying to determine the next execution time for a job.
func WithRetryInterval(retryInterval time.Duration) SchedulerOpt {
	return func(c *StdScheduler) error {
		c.opts.retryInterval = retryInterval
		return nil
	}
}

// WithMisfiredChan configures the channel to which misfired jobs are sent.
// A misfired job is a job that the scheduler was unable to execute according to
// its trigger schedule. If a channel is provided, misfired jobs are sent to it.
func WithMisfiredChan(misfiredChan chan ScheduledJob) SchedulerOpt {
	return func(c *StdScheduler) error {
		if misfiredChan == nil {
			return newIllegalArgumentError("misfiredChan is nil")
		}
		c.opts.misfiredChan = misfiredChan
		return nil
	}
}

// WithQueue configures the scheduler's job queue.
// Custom [JobQueue] and [sync.Locker] implementations can be provided to manage scheduled
// jobs which allows for persistent storage in distributed mode.
// A standard in-memory queue and a [sync.Mutex] are used by default.
func WithQueue(queue JobQueue, queueLocker sync.Locker) SchedulerOpt {
	return func(c *StdScheduler) error {
		if queue == nil {
			return newIllegalArgumentError("queue is nil")
		}
		if queueLocker == nil {
			return newIllegalArgumentError("queueLocker is nil")
		}
		c.queue = queue
		c.queueLocker = queueLocker
		return nil
	}
}

// WithLogger configures the logger used by the scheduler for logging messages.
// This enables the use of a custom logger implementation that satisfies the
// [logger.Logger] interface.
func WithLogger(logger logger.Logger) SchedulerOpt {
	return func(c *StdScheduler) error {
		if logger == nil {
			return newIllegalArgumentError("logger is nil")
		}
		c.logger = logger
		return nil
	}
}

// NewStdScheduler returns a new [StdScheduler] configured using the provided
// functional options.
//
// The following options are available for configuring the scheduler:
//
//   - WithBlockingExecution()
//   - WithWorkerLimit(workerLimit int)
//   - WithOutdatedThreshold(outdatedThreshold time.Duration)
//   - WithRetryInterval(retryInterval time.Duration)
//   - WithMisfiredChan(misfiredChan chan ScheduledJob)
//   - WithQueue(queue JobQueue, queueLocker sync.Locker)
//   - WithLogger(logger logger.Logger)
//
// Example usage:
//
//	scheduler, err := quartz.NewStdScheduler(
//		quartz.WithOutdatedThreshold(time.Second),
//		quartz.WithLogger(myLogger),
//	)
func NewStdScheduler(opts ...SchedulerOpt) (Scheduler, error) {
	// default scheduler configuration
	config := schedulerConfig{
		outdatedThreshold: 100 * time.Millisecond,
		retryInterval:     100 * time.Millisecond,
	}

	// initialize the scheduler with default values
	scheduler := &StdScheduler{
		interrupt:   make(chan struct{}, 1),
		feeder:      make(chan ScheduledJob),
		dispatch:    make(chan ScheduledJob),
		queue:       NewJobQueue(),
		queueLocker: &sync.Mutex{},
		opts:        config,
		logger:      logger.NoOpLogger{},
	}

	// apply functional options to configure the scheduler
	for _, opt := range opts {
		if err := opt(scheduler); err != nil {
			return nil, err
		}
	}

	return scheduler, nil
}

// ScheduleJob schedules a Job using the provided Trigger.
func (sched *StdScheduler) ScheduleJob(
	jobDetail *JobDetail,
	trigger Trigger,
) error {
	if jobDetail == nil {
		return newIllegalArgumentError("jobDetail is nil")
	}
	if jobDetail.jobKey == nil {
		return newIllegalArgumentError("jobDetail.jobKey is nil")
	}
	if jobDetail.jobKey.name == "" {
		return newIllegalArgumentError("empty key name is not allowed")
	}
	if trigger == nil {
		return newIllegalArgumentError("trigger is nil")
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

	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	if err = sched.queue.Push(toSchedule); err == nil {
		sched.logger.Debug("Successfully added job", "key", jobDetail.jobKey.String())
		if sched.IsStarted() {
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
		sched.logger.Info("Scheduler is already running")
		return
	}

	ctx, sched.cancel = context.WithCancel(ctx)
	go func() { <-ctx.Done(); sched.Stop() }()

	// start scheduler execution loop
	sched.wg.Add(1)
	go sched.startExecutionLoop(ctx)

	// starts worker pool if configured
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
	sched.mtx.RLock()
	defer sched.mtx.RUnlock()

	return sched.started
}

// GetJobKeys returns the keys of scheduled jobs.
// For a job key to be returned, the job must satisfy all of the matchers specified.
// Given no matchers, it returns the keys of all scheduled jobs.
func (sched *StdScheduler) GetJobKeys(matchers ...Matcher[ScheduledJob]) ([]*JobKey, error) {
	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

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
	if jobKey == nil {
		return nil, newIllegalArgumentError("jobKey is nil")
	}

	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	return sched.queue.Get(jobKey)
}

// DeleteJob removes the Job with the specified key if present.
func (sched *StdScheduler) DeleteJob(jobKey *JobKey) error {
	if jobKey == nil {
		return newIllegalArgumentError("jobKey is nil")
	}

	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	_, err := sched.queue.Remove(jobKey)
	if err == nil {
		sched.logger.Debug("Successfully deleted job", "key", jobKey.String())
		if sched.IsStarted() {
			sched.Reset()
		}
	}
	return err
}

// PauseJob suspends the job with the specified key from being
// executed by the scheduler.
func (sched *StdScheduler) PauseJob(jobKey *JobKey) error {
	if jobKey == nil {
		return newIllegalArgumentError("jobKey is nil")
	}

	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	job, err := sched.queue.Get(jobKey)
	if err != nil {
		return err
	}
	if job.JobDetail().opts.Suspended {
		return newIllegalStateError(ErrJobIsSuspended)
	}

	job, err = sched.queue.Remove(jobKey)
	if err == nil {
		job.JobDetail().opts.Suspended = true
		paused := &scheduledJob{
			job:      job.JobDetail(),
			trigger:  job.Trigger(),
			priority: int64(math.MaxInt64),
		}
		if err = sched.queue.Push(paused); err == nil {
			sched.logger.Debug("Successfully paused job", "key", jobKey.String())
			if sched.IsStarted() {
				sched.Reset()
			}
		}
	}
	return err
}

// ResumeJob restarts the suspended job with the specified key.
func (sched *StdScheduler) ResumeJob(jobKey *JobKey) error {
	if jobKey == nil {
		return newIllegalArgumentError("jobKey is nil")
	}

	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	job, err := sched.queue.Get(jobKey)
	if err != nil {
		return err
	}
	if !job.JobDetail().opts.Suspended {
		return newIllegalStateError(ErrJobIsActive)
	}

	job, err = sched.queue.Remove(jobKey)
	if err == nil {
		job.JobDetail().opts.Suspended = false
		var nextRunTime int64
		nextRunTime, err = job.Trigger().NextFireTime(NowNano())
		if err != nil {
			return err
		}
		resumed := &scheduledJob{
			job:      job.JobDetail(),
			trigger:  job.Trigger(),
			priority: nextRunTime,
		}
		if err = sched.queue.Push(resumed); err == nil {
			sched.logger.Debug("Successfully resumed job", "key", jobKey.String())
			if sched.IsStarted() {
				sched.Reset()
			}
		}
	}
	return err
}

// Clear removes all of the scheduled jobs.
func (sched *StdScheduler) Clear() error {
	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	// reset the job queue
	err := sched.queue.Clear()
	if err == nil {
		sched.logger.Debug("Successfully cleared job queue")
		if sched.IsStarted() {
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
		sched.logger.Info("Scheduler is not running")
		return
	}

	sched.logger.Info("Closing the scheduler")
	sched.cancel()
	sched.started = false
}

func (sched *StdScheduler) startExecutionLoop(ctx context.Context) {
	defer sched.wg.Done()
	const maxTimerDuration = time.Duration(1<<63 - 1)
	timer := time.NewTimer(maxTimerDuration)
	for {
		queueSize, err := sched.queue.Size()
		switch {
		case err != nil:
			sched.logger.Error("Failed to fetch queue size", "error", err)
			timer.Reset(sched.opts.retryInterval)
		case queueSize == 0:
			sched.logger.Trace("Queue is empty")
			timer.Reset(maxTimerDuration)
		default:
			timer.Reset(sched.calculateNextTick())
		}
		select {
		case <-timer.C:
			sched.logger.Trace("Tick")
			sched.executeAndReschedule(ctx)

		case <-sched.interrupt:
			sched.logger.Trace("Interrupted waiting for next tick")
			timer.Stop()

		case <-ctx.Done():
			sched.logger.Info("Exit the execution loop")
			timer.Stop()
			return
		}
	}
}

func (sched *StdScheduler) startWorkers(ctx context.Context) {
	if !sched.opts.blockingExecution && sched.opts.workerLimit > 0 {
		sched.logger.Debug("Starting scheduler workers", "n", sched.opts.workerLimit)
		for i := 0; i < sched.opts.workerLimit; i++ {
			sched.wg.Add(1)
			go func() {
				defer sched.wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case scheduled := <-sched.dispatch:
						sched.executeWithRetries(ctx, scheduled.JobDetail())
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
			sched.logger.Debug("Queue is empty")
			return nextTickDuration
		}
		sched.logger.Error("Failed to calculate next tick", "error", err)
		return sched.opts.retryInterval
	}

	nextRunTime := scheduledJob.NextRunTime()
	now := NowNano()
	if nextRunTime > now {
		nextTickDuration = time.Duration(nextRunTime - now)
	}
	sched.logger.Trace("Next tick", "job", scheduledJob.JobDetail().jobKey.String(),
		"after", nextTickDuration)

	return nextTickDuration
}

func (sched *StdScheduler) executeAndReschedule(ctx context.Context) {
	// fetch a job for processing
	scheduled, valid := sched.fetchAndReschedule()

	// execute the job
	if valid {
		sched.logger.Debug("Job is about to be executed",
			"key", scheduled.JobDetail().jobKey.String())
		switch {
		case sched.opts.blockingExecution:
			sched.executeWithRetries(ctx, scheduled.JobDetail())
		case sched.opts.workerLimit > 0:
			select {
			case sched.dispatch <- scheduled:
			case <-ctx.Done():
				return
			}
		default:
			sched.wg.Add(1)
			go func() {
				defer sched.wg.Done()
				sched.executeWithRetries(ctx, scheduled.JobDetail())
			}()
		}
	}
}

func (sched *StdScheduler) executeWithRetries(ctx context.Context, jobDetail *JobDetail) {
	// recover from unhandled panics that may occur during job execution
	defer func() {
		if err := recover(); err != nil {
			sched.logger.Error("Job panicked", "key", jobDetail.jobKey.String(),
				"error", err)
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
		sched.logger.Trace("Job retry", "key", jobDetail.jobKey.String(), "attempt", i)
		err = jobDetail.job.Execute(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		sched.logger.Warn("Job terminated", "key", jobDetail.jobKey.String(), "error", err)
	}
}

func (sched *StdScheduler) validateJob(job ScheduledJob) (bool, func() (int64, error)) {
	if job.JobDetail().opts.Suspended {
		return false, func() (int64, error) { return math.MaxInt64, nil }
	}

	now := NowNano()
	if job.NextRunTime() < now-sched.opts.outdatedThreshold.Nanoseconds() {
		duration := time.Duration(now - job.NextRunTime())
		sched.logger.Info("Job is outdated", "key", job.JobDetail().jobKey.String(),
			"duration", duration)
		select {
		case sched.opts.misfiredChan <- job:
		default:
		}
		return false, func() (int64, error) { return job.Trigger().NextFireTime(now) }
	} else if job.NextRunTime() > now {
		sched.logger.Debug("Job is not due to run yet",
			"key", job.JobDetail().jobKey.String())
		return false, func() (int64, error) { return job.NextRunTime(), nil }
	}

	return true, func() (int64, error) {
		return job.Trigger().NextFireTime(job.NextRunTime())
	}
}

func (sched *StdScheduler) fetchAndReschedule() (ScheduledJob, bool) {
	sched.queueLocker.Lock()
	defer sched.queueLocker.Unlock()

	// fetch a job for processing
	job, err := sched.queue.Pop()
	if err != nil {
		if errors.Is(err, ErrQueueEmpty) {
			sched.logger.Debug("Queue is empty")
		} else {
			sched.logger.Error("Failed to fetch a job from the queue", "error", err)
		}
		return nil, false
	}

	// validate the job
	valid, nextRunTimeExtractor := sched.validateJob(job)

	// calculate next run time for the job
	nextRunTime, err := nextRunTimeExtractor()
	if err != nil {
		sched.logger.Info("Job exited the execution loop",
			"key", job.JobDetail().jobKey.String(), "error", err)
		return job, valid
	}

	// reschedule the job
	toSchedule := &scheduledJob{
		job:      job.JobDetail(),
		trigger:  job.Trigger(),
		priority: nextRunTime,
	}
	if err := sched.queue.Push(toSchedule); err != nil {
		sched.logger.Error("Failed to reschedule job",
			"key", toSchedule.JobDetail().jobKey.String(), "error", err)
	} else {
		sched.logger.Trace("Successfully rescheduled job",
			"key", toSchedule.JobDetail().jobKey.String())
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
