package quartz

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

type Job interface {
	Execute()
	Description() string
}

type Scheduler interface {
	Start()
	ScheduleJob(job Job, trigger Trigger) error
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
	nextRunTime, err := trigger.NextFireTime(time.Now().UnixNano())
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
	//fetch item
	sched.Lock()
	item := heap.Pop(sched.Queue).(*Item)
	sched.Unlock()
	//execute Job
	go item.Job.Execute()
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
			select {
			case sched.interrupt <- struct{}{}:
			default:
			}
			sched.Unlock()
		case <-sched.exit:
			fmt.Println("Exit feed reader")
			return
		}
	}
}

func parkTime(ts int64) int64 {
	now := time.Now().UnixNano()
	if ts > now {
		return ts - now
	}
	return 0
}
