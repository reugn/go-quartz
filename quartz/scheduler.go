package quartz

import (
	"container/heap"
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
	// Start starts the scheduler.
	Start()

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
}

// StdScheduler implements the quartz.Scheduler interface.
type StdScheduler struct {
	sync.Mutex
	queue     *priorityQueue
	interrupt chan struct{}
	exit      chan struct{}
	feeder    chan *item
	dispatch  chan *item
	started   bool
	opts      StdSchedulerOptions
}

type StdSchedulerOptions struct {
	// When true, the scheduler will run jobs synchronously,
	// waiting for each exceution instance of the job to return
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
}

// Verify StdScheduler satisfies the Scheduler interface.
var _ Scheduler = (*StdScheduler)(nil)

// NewStdScheduler returns a new StdScheduler with the default configuration.
func NewStdScheduler() Scheduler {
	return NewStdSchedulerWithOptions(StdSchedulerOptions{})
}

// NewStdSchedulerWithOptions returns a new StdScheduler configured as specified.
func NewStdSchedulerWithOptions(opts StdSchedulerOptions) *StdScheduler {
	return &StdScheduler{
		queue:     &priorityQueue{},
		interrupt: make(chan struct{}, 1),
		exit:      nil,
		feeder:    make(chan *item),
		dispatch:  make(chan *item),
		opts:      opts,
	}
}

// ScheduleJob schedules a Job using a specified Trigger.
func (sched *StdScheduler) ScheduleJob(job Job, trigger Trigger) error {
	nextRunTime, err := trigger.NextFireTime(NowNano())
	if err != nil {
		return err
	}

	sched.feeder <- &item{
		Job:      job,
		Trigger:  trigger,
		priority: nextRunTime,
		index:    0,
	}

	return nil
}

// Start starts the StdScheduler execution loop.
func (sched *StdScheduler) Start() {
	sched.Lock()
	defer sched.Unlock()

	if sched.started {
		return
	}

	// reset the exit channel
	sched.exit = make(chan struct{})

	// start the feed reader
	go sched.startFeedReader()

	// start scheduler execution loop
	go sched.startExecutionLoop()

	// starts worker pool when WorkerLimit is > 0
	sched.startWorkers()

	sched.started = true
}

// IsStarted determines whether the scheduler has been started.
func (sched *StdScheduler) IsStarted() bool {
	return sched.started
}

// GetJobKeys returns the keys of all of the scheduled jobs.
func (sched *StdScheduler) GetJobKeys() []int {
	sched.Lock()
	defer sched.Unlock()

	keys := make([]int, 0, sched.queue.Len())
	for _, item := range *sched.queue {
		keys = append(keys, item.Job.Key())
	}

	return keys
}

// GetScheduledJob returns the ScheduledJob with the specified key.
func (sched *StdScheduler) GetScheduledJob(key int) (*ScheduledJob, error) {
	sched.Lock()
	defer sched.Unlock()

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
	sched.Lock()
	defer sched.Unlock()

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
	sched.Lock()
	defer sched.Unlock()

	// reset the job queue
	sched.queue = &priorityQueue{}
}

// Stop exits the StdScheduler execution loop.
func (sched *StdScheduler) Stop() {
	sched.Lock()
	defer sched.Unlock()

	if !sched.started {
		return
	}

	log.Printf("Closing the StdScheduler.")
	close(sched.exit)

	sched.started = false
}

func (sched *StdScheduler) startExecutionLoop() {
	for {
		if sched.queueLen() == 0 {
			select {
			case <-sched.interrupt:
			case <-sched.exit:
				log.Printf("Exit the empty execution loop.")
				return
			}
		} else {
			t := time.NewTimer(sched.calculateNextTick())
			select {
			case <-t.C:
				sched.executeAndReschedule()

			case <-sched.interrupt:
				t.Stop()

			case <-sched.exit:
				log.Printf("Exit the execution loop.")
				t.Stop()
				return
			}
		}
	}
}

func (sched *StdScheduler) startWorkers() {
	if sched.opts.WorkerLimit > 0 {
		for i := 0; i < sched.opts.WorkerLimit; i++ {
			go func() {
				for {
					select {
					// case <-ctx.Done():
					//	return
					case <-sched.exit:
						return
					case item := <-sched.dispatch:
						item.Job.Execute()
					}
				}
			}()
		}
	}
}

func (sched *StdScheduler) queueLen() int {
	sched.Lock()
	defer sched.Unlock()

	return sched.queue.Len()
}

func (sched *StdScheduler) calculateNextTick() time.Duration {
	sched.Lock()
	var interval int64
	if sched.queue.Len() > 0 {
		interval = parkTime(sched.queue.Head().priority)
	}
	sched.Unlock()

	return time.Duration(interval)
}

func (sched *StdScheduler) executeAndReschedule() {
	// return if the job queue is empty
	if sched.queueLen() == 0 {
		return
	}

	// fetch an item
	sched.Lock()
	item := heap.Pop(sched.queue).(*item)
	sched.Unlock()

	// execute the Job
	if !isOutdated(item.priority) {
		if sched.opts.BlockingExecution {
			item.Job.Execute()
		} else if sched.opts.WorkerLimit > 0 {
			select {
			case sched.dispatch <- item:
			case <-sched.exit:
				return
			}
		} else {
			go item.Job.Execute()
		}
	}

	// reschedule the Job
	nextRunTime, err := item.Trigger.NextFireTime(item.priority)
	if err != nil {
		log.Printf("The Job '%s' got out the execution loop.", item.Job.Description())
		return
	}
	item.priority = nextRunTime
	sched.feeder <- item
}

func (sched *StdScheduler) startFeedReader() {
	for {
		select {
		case item := <-sched.feeder:
			sched.Lock()
			heap.Push(sched.queue, item)
			sched.reset()
			sched.Unlock()

		case <-sched.exit:
			log.Printf("Exit the feed reader.")
			return
		}
	}
}

func (sched *StdScheduler) reset() {
	select {
	case sched.interrupt <- struct{}{}:
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
