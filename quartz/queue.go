package quartz

import (
	"container/heap"
	"fmt"
	"sync"
)

// scheduledJob represents a scheduled job.
// It implements the ScheduledJob interface.
type scheduledJob struct {
	job      *JobDetail
	trigger  Trigger
	priority int64 // job priority, backed by its next run time.
	index    int   // maintained by the heap.Interface methods.
}

var _ ScheduledJob = (*scheduledJob)(nil)

// Job returns the scheduled job instance.
func (scheduled *scheduledJob) JobDetail() *JobDetail {
	return scheduled.job
}

// Trigger returns the trigger associated with the scheduled job.
func (scheduled *scheduledJob) Trigger() Trigger {
	return scheduled.trigger
}

// NextRunTime returns the next run epoch time for the scheduled job.
func (scheduled *scheduledJob) NextRunTime() int64 {
	return scheduled.priority
}

// JobQueue represents the job queue used by the scheduler.
// The default jobQueue implementation uses an in-memory priority queue that orders
// scheduled jobs by their next execution time, when the job with the closest time
// being removed and returned first.
// An alternative implementation can be provided for customization, e.g. to support
// persistent storage.
// The implementation is required to be thread safe.
type JobQueue interface {
	// Push inserts a new scheduled job to the queue.
	// This method is also used by the Scheduler to reschedule existing jobs that
	// have been dequeued for execution.
	Push(job ScheduledJob) error

	// Pop removes and returns the next to run scheduled job from the queue.
	Pop() (ScheduledJob, error)

	// Head returns the first scheduled job without removing it from the queue.
	Head() (ScheduledJob, error)

	// Get returns the scheduled job with the specified key without removing it
	// from the queue.
	Get(jobKey *JobKey) (ScheduledJob, error)

	// Remove removes and returns the scheduled job with the specified key.
	Remove(jobKey *JobKey) (ScheduledJob, error)

	// ScheduledJobs returns the slice of all scheduled jobs in the queue.
	ScheduledJobs() []ScheduledJob

	// Size returns the size of the job queue.
	Size() int

	// Clear clears the job queue.
	Clear() error
}

// priorityQueue implements the heap.Interface.
type priorityQueue []*scheduledJob

var _ heap.Interface = (*priorityQueue)(nil)

// Len returns the priorityQueue length.
func (pq priorityQueue) Len() int {
	return len(pq)
}

// Less is the items less comparator.
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

// Swap exchanges the indexes of the items.
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push implements the heap.Interface.Push.
// Adds an element at index Len().
func (pq *priorityQueue) Push(element interface{}) {
	index := len(*pq)
	item := element.(*scheduledJob)
	item.index = index
	*pq = append(*pq, item)
}

// Pop implements the heap.Interface.Pop.
// Removes and returns the element at Len() - 1.
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// jobQueue implements the JobQueue interface by using an in-memory
// priority queue as the storage layer.
type jobQueue struct {
	mtx      sync.Mutex
	delegate priorityQueue
}

var _ JobQueue = (*jobQueue)(nil)

// newJobQueue initializes and returns an empty jobQueue.
func newJobQueue() *jobQueue {
	return &jobQueue{
		delegate: priorityQueue{},
	}
}

// Push inserts a new scheduled job to the queue.
// This method is also used by the Scheduler to reschedule existing jobs that
// have been dequeued for execution.
func (jq *jobQueue) Push(job ScheduledJob) error {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	scheduledJobs := jq.scheduledJobs()
	for i, scheduled := range scheduledJobs {
		if scheduled.JobDetail().jobKey.Equals(job.JobDetail().jobKey) {
			if job.JobDetail().opts.Replace {
				heap.Remove(&jq.delegate, i)
				break
			}
			return illegalStateError(fmt.Sprintf("job with the key %s already exists",
				job.JobDetail().jobKey))
		}
	}
	heap.Push(&jq.delegate, job)
	return nil
}

// Pop removes and returns the next scheduled job from the queue.
func (jq *jobQueue) Pop() (ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	if len(jq.delegate) == 0 {
		return nil, ErrQueueEmpty
	}
	return heap.Pop(&jq.delegate).(ScheduledJob), nil
}

// Head returns the first scheduled job without removing it from the queue.
func (jq *jobQueue) Head() (ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	if len(jq.delegate) == 0 {
		return nil, ErrQueueEmpty
	}
	return jq.delegate[0], nil
}

// Get returns the scheduled job with the specified key without removing it
// from the queue.
func (jq *jobQueue) Get(jobKey *JobKey) (ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	for _, scheduled := range jq.delegate {
		if scheduled.JobDetail().jobKey.Equals(jobKey) {
			return scheduled, nil
		}
	}
	return nil, jobNotFoundError(jobKey.String())
}

// Remove removes and returns the scheduled job with the specified key.
func (jq *jobQueue) Remove(jobKey *JobKey) (ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	scheduledJobs := jq.scheduledJobs()
	for i, scheduled := range scheduledJobs {
		if scheduled.JobDetail().jobKey.Equals(jobKey) {
			return heap.Remove(&jq.delegate, i).(ScheduledJob), nil
		}
	}
	return nil, jobNotFoundError(jobKey.String())
}

// ScheduledJobs returns the slice of all scheduled jobs in the queue.
func (jq *jobQueue) ScheduledJobs() []ScheduledJob {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	return jq.scheduledJobs()
}

func (jq *jobQueue) scheduledJobs() []ScheduledJob {
	scheduledJobs := make([]ScheduledJob, len(jq.delegate))
	for i, job := range jq.delegate {
		scheduledJobs[i] = ScheduledJob(job)
	}
	return scheduledJobs
}

// Size returns the size of the job queue.
func (jq *jobQueue) Size() int {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	return len(jq.delegate)
}

// Clear clears the job queue.
func (jq *jobQueue) Clear() error {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()
	jq.delegate = priorityQueue{}
	return nil
}
