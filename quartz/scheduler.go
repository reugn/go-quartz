package quartz

import (
	"container/heap"
	"errors"
	"log"
	"sync"
	"time"
)

// ScheduledJob wraps the scheduled job with the metadata.
type ScheduledJob struct {
	Job                Job
	TriggerDescription string
	NextRunTime        int64
}

// A Scheduler is the Jobs orchestrator.
// Schedulers responsible for executing Jobs when their associated Triggers fire (when their scheduled time arrives).
type Scheduler interface {
	// start the scheduler
	Start()
	// whether the scheduler has been started
	IsStarted() bool
	// schedule the job with the specified trigger
	ScheduleJob(job Job, trigger Trigger) error
	// get keys of all of the scheduled jobs
	GetJobKeys() []int
	// get the scheduled job metadata
	GetScheduledJob(key int) (*ScheduledJob, error)
	// remove the job from the execution queue
	DeleteJob(key int) error
	// clear all the scheduled jobs
	Clear()
	// shutdown the scheduler
	Stop()
}

// StdScheduler implements the quartz.Scheduler interface.
type StdScheduler struct {
	sync.Mutex
	Queue     *PriorityQueue
	interrupt chan struct{}
	exit      chan struct{}
	feeder    chan *Item
	started   bool
}

// NewStdScheduler returns a new StdScheduler.
func NewStdScheduler() *StdScheduler {
	return &StdScheduler{
		Queue:     &PriorityQueue{},
		interrupt: make(chan struct{}, 1),
		exit:      nil,
		feeder:    make(chan *Item)}
}

// ScheduleJob uses the specified Trigger to schedule the Job.
func (sched *StdScheduler) ScheduleJob(job Job, trigger Trigger) error {
	nextRunTime, err := trigger.NextFireTime(NowNano())

	if err == nil {
		sched.feeder <- &Item{
			job,
			trigger,
			nextRunTime,
			0}
		return nil
	}

	return err
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

	sched.started = true
}

// IsStarted states whether the scheduler has been started.
func (sched *StdScheduler) IsStarted() bool {
	return sched.started
}

// GetJobKeys returns the keys of all of the scheduled jobs.
func (sched *StdScheduler) GetJobKeys() []int {
	sched.Lock()
	defer sched.Unlock()

	keys := make([]int, 0, sched.Queue.Len())
	for _, item := range *sched.Queue {
		keys = append(keys, item.Job.Key())
	}

	return keys
}

// GetScheduledJob returns the ScheduledJob by the unique key.
func (sched *StdScheduler) GetScheduledJob(key int) (*ScheduledJob, error) {
	sched.Lock()
	defer sched.Unlock()

	for _, item := range *sched.Queue {
		if item.Job.Key() == key {
			return &ScheduledJob{
				item.Job,
				item.Trigger.Description(),
				item.priority,
			}, nil
		}
	}

	return nil, errors.New("No Job with the given Key found")
}

// DeleteJob removes the job for the specified key from the StdScheduler if present.
func (sched *StdScheduler) DeleteJob(key int) error {
	sched.Lock()
	defer sched.Unlock()

	for i, item := range *sched.Queue {
		if item.Job.Key() == key {
			sched.Queue.Remove(i)
			return nil
		}
	}

	return errors.New("No Job with the given Key found")
}

// Clear removes all of the scheduled jobs.
func (sched *StdScheduler) Clear() {
	sched.Lock()
	defer sched.Unlock()

	// reset the jobs queue
	sched.Queue = &PriorityQueue{}
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
				continue
			case <-sched.exit:
				log.Printf("Exit the execution loop.")
				t.Stop()
				return
			}
		}
	}
}

func (sched *StdScheduler) queueLen() int {
	sched.Lock()
	defer sched.Unlock()

	return sched.Queue.Len()
}

func (sched *StdScheduler) calculateNextTick() time.Duration {
	sched.Lock()
	var interval int64
	if sched.Queue.Len() > 0 {
		interval = parkTime(sched.Queue.Head().priority)
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
	item := heap.Pop(sched.Queue).(*Item)
	sched.Unlock()

	// execute the Job
	if !isOutdated(item.priority) {
		go item.Job.Execute()
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
			heap.Push(sched.Queue, item)
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
