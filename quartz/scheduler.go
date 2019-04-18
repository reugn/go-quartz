package quartz

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Job interface {
	Execute()
	Description() string
	Key() int
}

type ScheduledJob struct {
	Job                Job
	TriggerDescription string
	NextRunTime        int64
}

type Scheduler interface {
	//start scheduler
	Start()
	//schedule Job with given Trigger
	ScheduleJob(job Job, trigger Trigger) error
	//get all scheduled Job keys
	GetJobKeys() []int
	//get scheduled Job metadata
	GetScheduledJob(key int) (*ScheduledJob, error)
	//remove Job from execution queue
	DeleteJob(key int) error
	//clear all scheduled jobs
	Clear()
	//shutdown scheduler
	Stop()
}

//implements quartz.Scheduler interface
type StdScheduler struct {
	sync.Mutex
	Queue     *PriorityQueue
	interrupt chan interface{}
	exit      chan interface{}
	feeder    chan *Item
}

func NewStdScheduler() *StdScheduler {
	return &StdScheduler{
		Queue:     &PriorityQueue{},
		interrupt: make(chan interface{}),
		exit:      nil,
		feeder:    make(chan *Item)}
}

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

func (sched *StdScheduler) Start() {
	//reset exit channel
	sched.exit = make(chan interface{})
	//start feed reader
	go sched.startFeedReader()
	//start scheduler execution loop
	go sched.startExecutionLoop()
}

func (sched *StdScheduler) GetJobKeys() []int {
	sched.Lock()
	defer sched.Unlock()
	keys := make([]int, 0, sched.Queue.Len())
	for _, item := range *sched.Queue {
		keys = append(keys, item.Job.Key())
	}
	return keys
}

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
	return nil, errors.New("No Job with given Key found")
}

func (sched *StdScheduler) DeleteJob(key int) error {
	sched.Lock()
	defer sched.Unlock()
	for _, item := range *sched.Queue {
		if item.Job.Key() == key {
			sched.Queue.Pop()
			sched.reset()
			return nil
		}
	}
	return errors.New("No Job with given Key found")
}

func (sched *StdScheduler) Clear() {
	sched.Lock()
	defer sched.Unlock()
	sched.Queue = &PriorityQueue{}
}

func (sched *StdScheduler) Stop() {
	fmt.Println("Closing scheduler")
	close(sched.exit)
}

func (sched *StdScheduler) startExecutionLoop() {
	for {
		if sched.queueLen() == 0 {
			select {
			case <-sched.interrupt:
			case <-sched.exit:
				return
			}
		} else {
			tick := sched.calculateNextTick()
			select {
			case <-tick:
				sched.executeAndReschedule()
			case <-sched.interrupt:
				continue
			case <-sched.exit:
				fmt.Println("Exit execution loop")
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

func (sched *StdScheduler) calculateNextTick() <-chan time.Time {
	sched.Lock()
	ts := sched.Queue.Head().priority
	sched.Unlock()
	return time.After(time.Duration(parkTime(ts)))
}

func (sched *StdScheduler) executeAndReschedule() {
	if sched.queueLen() == 0 {
		return
	}
	//fetch item
	sched.Lock()
	item := heap.Pop(sched.Queue).(*Item)
	sched.Unlock()
	//execute Job
	if !isOutdated(item.priority) {
		go item.Job.Execute()
	}
	//reschedule Job
	nextRunTime, err := item.Trigger.NextFireTime(item.priority)
	if err == nil {
		item.priority = nextRunTime
		sched.feeder <- item
	}
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
			fmt.Println("Exit feed reader")
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
