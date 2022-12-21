package quartz

import (
	"container/heap"
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// ScheduledJob wraps a scheduled Job with its metadata.
type ScheduledJob struct {
	Job                Job
	TriggerDescription string
	NextRunTime        int64
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
	ScheduleJob(job Job, trigger Trigger) error

	// GetJobKeys returns the keys of all of the scheduled jobs.
	GetJobKeys() []int

	// GetScheduledJob returns the scheduled job with the specified key.
	GetScheduledJob(key int) (*ScheduledJob, error)

	// DeleteJob removes the job with the specified key from the Scheduler's execution queue.
	DeleteJob(key int) error

	// Clear removes all of the scheduled jobs.
	Clear()

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
	queue     *priorityQueue
	interrupt chan struct{}
	signal    chan struct{}
	feeder    chan *item
	started   bool
	cancel    context.CancelFunc
}

// Verify StdScheduler satisfies the Scheduler interface.
var _ Scheduler = (*StdScheduler)(nil)

// NewStdScheduler returns a new StdScheduler.
func NewStdScheduler() *StdScheduler {
	return &StdScheduler{
		queue:     &priorityQueue{},
		interrupt: make(chan struct{}, 1),
		cancel:    func() {},
		feeder:    make(chan *item),
		signal:    make(chan struct{}),
	}
}

// ScheduleJob schedules a Job using a specified Trigger.
func (sched *StdScheduler) ScheduleJob(job Job, trigger Trigger) error {
	nextRunTime, err := trigger.NextFireTime(NowNano())
	if err != nil {
		return err
	}

	select {
	case sched.feeder <- &item{
		Job:      job,
		Trigger:  trigger,
		priority: nextRunTime,
		index:    0,
	}:
		return nil
	case <-sched.signal:
		return context.Canceled
	}
}

// Start starts the StdScheduler execution loop.
func (sched *StdScheduler) Start(ctx context.Context) {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	if sched.started {
		return
	}

	ctx, sched.cancel = context.WithCancel(ctx)
	go func() { <-ctx.Done(); sched.Stop() }()
	// start the feed reader
	go sched.startFeedReader(ctx)

	// start scheduler execution loop
	go sched.startExecutionLoop(ctx)

	sched.started = true
	sched.signal = make(chan struct{})
}

func (sched *StdScheduler) Wait(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-sched.signal:
	}
}

// IsStarted determines whether the scheduler has been started.
func (sched *StdScheduler) IsStarted() bool {
	return sched.started
}

// GetJobKeys returns the keys of all of the scheduled jobs.
func (sched *StdScheduler) GetJobKeys() []int {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	keys := make([]int, 0, sched.queue.Len())
	for _, item := range *sched.queue {
		keys = append(keys, item.Job.Key())
	}

	return keys
}

// GetScheduledJob returns the ScheduledJob with the specified key.
func (sched *StdScheduler) GetScheduledJob(key int) (*ScheduledJob, error) {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	for _, item := range *sched.queue {
		if item.Job.Key() == key {
			return &ScheduledJob{
				Job:                item.Job,
				TriggerDescription: item.Trigger.Description(),
				NextRunTime:        item.priority,
			}, nil
		}
	}

	return nil, errors.New("no Job with the given Key found")
}

// DeleteJob removes the Job with the specified key if present.
func (sched *StdScheduler) DeleteJob(key int) error {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	for i, item := range *sched.queue {
		if item.Job.Key() == key {
			sched.queue.Remove(i)
			return nil
		}
	}

	return errors.New("no Job with the given Key found")
}

// Clear removes all of the scheduled jobs.
func (sched *StdScheduler) Clear() {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	// reset the job queue
	sched.queue = &priorityQueue{}
}

// Stop exits the StdScheduler execution loop.
func (sched *StdScheduler) Stop() {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	if !sched.started {
		return
	}

	log.Printf("Closing the StdScheduler.")
	sched.cancel()
	sched.started = false
	close(sched.signal)
}

func (sched *StdScheduler) startExecutionLoop(ctx context.Context) {

	for {
		if sched.queueLen() == 0 {
			select {
			case <-sched.interrupt:
			case <-ctx.Done():
				log.Printf("Exit the empty execution loop.")
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
				log.Printf("Exit the execution loop.")
				t.Stop()
				return
			}
		}
	}
}

func (sched *StdScheduler) queueLen() int {
	sched.mtx.Lock()
	defer sched.mtx.Unlock()

	return sched.queue.Len()
}

func (sched *StdScheduler) calculateNextTick() time.Duration {
	var interval int64

	sched.mtx.Lock()
	defer sched.mtx.Unlock()
	if sched.queue.Len() > 0 {
		interval = parkTime(sched.queue.Head().priority)
	}

	return time.Duration(interval)
}

func (sched *StdScheduler) executeAndReschedule(ctx context.Context) {
	// return if the job queue is empty
	if sched.queueLen() == 0 {
		return
	}

	// fetch an item
	var it *item
	func() {
		sched.mtx.Lock()
		defer sched.mtx.Unlock()
		it = heap.Pop(sched.queue).(*item)
	}()

	// execute the Job
	if !isOutdated(it.priority) {
		go it.Job.Execute(ctx)
	}

	// reschedule the Job
	nextRunTime, err := it.Trigger.NextFireTime(it.priority)
	if err != nil {
		log.Printf("The Job '%s' got out the execution loop: %q", it.Job.Description(), err.Error())
		return
	}

	it.priority = nextRunTime
	select {
	case <-ctx.Done():
	case sched.feeder <- it:
	}
}

func (sched *StdScheduler) startFeedReader(ctx context.Context) {
	for {
		select {
		case item := <-sched.feeder:
			func() {
				sched.mtx.Lock()
				defer sched.mtx.Unlock()

				heap.Push(sched.queue, item)
				sched.reset(ctx)
			}()
		case <-ctx.Done():
			log.Printf("Exit the feed reader.")
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
