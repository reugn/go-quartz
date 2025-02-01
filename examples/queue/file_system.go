package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

const (
	dataFolder             = "./store"
	fileMode   fs.FileMode = 0744
)

func init() {
	quartz.Sep = "_"
}

//nolint:funlen
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 31*time.Second)
	defer cancel()

	stdLogger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix|log.Lshortfile)
	l := logger.NewSimpleLogger(stdLogger, logger.LevelTrace)

	if _, err := os.Stat(dataFolder); os.IsNotExist(err) {
		if err := os.Mkdir(dataFolder, fileMode); err != nil {
			l.Warn("Failed to create data folder", "error", err)
			return
		}
	}

	l.Info("Starting scheduler")
	jobQueue := newJobQueue(l)
	scheduler, err := quartz.NewStdScheduler(
		quartz.WithOutdatedThreshold(time.Second), // considering file system I/O latency
		quartz.WithQueue(jobQueue, &sync.Mutex{}),
		quartz.WithLogger(l),
	)
	if err != nil {
		l.Error("Failed to create scheduler", "error", err)
		return
	}

	scheduler.Start(ctx)

	jobQueueSize, err := jobQueue.Size()
	if err != nil {
		l.Error("Failed to fetch job queue size", "error", err)
		return
	}

	if jobQueueSize == 0 {
		l.Info("Scheduling new jobs")
		jobDetail1 := quartz.NewJobDetail(newPrintJob(5, l), quartz.NewJobKey("job1"))
		if err := scheduler.ScheduleJob(jobDetail1, quartz.NewSimpleTrigger(5*time.Second)); err != nil {
			l.Warn("Failed to schedule job", "key", jobDetail1.JobKey().String(), "error", err)
		}
		jobDetail2 := quartz.NewJobDetail(newPrintJob(10, l), quartz.NewJobKey("job2"))
		if err := scheduler.ScheduleJob(jobDetail2, quartz.NewSimpleTrigger(10*time.Second)); err != nil {
			l.Warn("Failed to schedule job", "key", jobDetail2.JobKey().String(), "error", err)
		}
	} else {
		l.Info("Job queue is not empty")
	}

	<-ctx.Done()

	scheduledJobs, err := jobQueue.ScheduledJobs(nil)
	if err != nil {
		l.Error("Failed to fetch scheduled jobs", "error", err)
		return
	}

	jobNames := make([]string, 0, len(scheduledJobs))
	for _, job := range scheduledJobs {
		jobNames = append(jobNames, job.JobDetail().JobKey().String())
	}

	l.Info("Jobs in queue", "names", jobNames)
}

// printJob
type printJob struct {
	seconds int
	logger  logger.Logger
}

var _ quartz.Job = (*printJob)(nil)

func newPrintJob(seconds int, logger logger.Logger) *printJob {
	return &printJob{
		seconds: seconds,
		logger:  logger,
	}
}

func (job *printJob) Execute(_ context.Context) error {
	job.logger.Info(fmt.Sprintf("PrintJob: %d", job.seconds))
	return nil
}

func (job *printJob) Description() string {
	return fmt.Sprintf("PrintJob%s%d", quartz.Sep, job.seconds)
}

// scheduledPrintJob
type scheduledPrintJob struct {
	jobDetail   *quartz.JobDetail
	trigger     quartz.Trigger
	nextRunTime int64
}

// serializedJob
type serializedJob struct {
	Job     string                   `json:"job"`
	JobKey  string                   `json:"job_key"`
	Options *quartz.JobDetailOptions `json:"job_options"`

	Trigger     string `json:"trigger"`
	NextRunTime int64  `json:"next_run_time"`
}

var _ quartz.ScheduledJob = (*scheduledPrintJob)(nil)

func (job *scheduledPrintJob) JobDetail() *quartz.JobDetail { return job.jobDetail }
func (job *scheduledPrintJob) Trigger() quartz.Trigger      { return job.trigger }
func (job *scheduledPrintJob) NextRunTime() int64           { return job.nextRunTime }

// marshal returns the JSON encoding of the job.
func marshal(job quartz.ScheduledJob) ([]byte, error) {
	var serialized serializedJob
	serialized.Job = job.JobDetail().Job().Description()
	serialized.JobKey = job.JobDetail().JobKey().String()
	serialized.Options = job.JobDetail().Options()
	serialized.Trigger = job.Trigger().Description()
	serialized.NextRunTime = job.NextRunTime()

	return json.Marshal(serialized)
}

// unmarshal parses the JSON-encoded job.
func unmarshal(encoded []byte, l logger.Logger) (quartz.ScheduledJob, error) {
	var serialized serializedJob
	if err := json.Unmarshal(encoded, &serialized); err != nil {
		return nil, err
	}

	jobVals := strings.Split(serialized.Job, quartz.Sep)
	i, err := strconv.Atoi(jobVals[1])
	if err != nil {
		return nil, err
	}

	job := newPrintJob(i, l) // assuming we know the job type
	jobKeyVals := strings.Split(serialized.JobKey, quartz.Sep)
	jobKey := quartz.NewJobKeyWithGroup(jobKeyVals[1], jobKeyVals[0])
	jobDetail := quartz.NewJobDetailWithOptions(job, jobKey, serialized.Options)
	triggerOpts := strings.Split(serialized.Trigger, quartz.Sep)
	interval, _ := time.ParseDuration(triggerOpts[1])
	trigger := quartz.NewSimpleTrigger(interval) // assuming we know the trigger type

	return &scheduledPrintJob{
		jobDetail:   jobDetail,
		trigger:     trigger,
		nextRunTime: serialized.NextRunTime,
	}, nil
}

// jobQueue implements the quartz.JobQueue interface, using the file system
// as the persistence layer.
type jobQueue struct {
	mtx    sync.Mutex
	logger logger.Logger
}

var _ quartz.JobQueue = (*jobQueue)(nil)

// newJobQueue initializes and returns an empty jobQueue.
func newJobQueue(logger logger.Logger) *jobQueue {
	return &jobQueue{logger: logger}
}

// Push inserts a new scheduled job to the queue.
// This method is also used by the Scheduler to reschedule existing jobs that
// have been dequeued for execution.
func (jq *jobQueue) Push(job quartz.ScheduledJob) error {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Push job", "key", job.JobDetail().JobKey().String())
	serialized, err := marshal(job)
	if err != nil {
		return err
	}
	if err = os.WriteFile(fmt.Sprintf("%s/%d", dataFolder, job.NextRunTime()),
		serialized, fileMode); err != nil {
		jq.logger.Error("Failed to write job", "error", err)
		return err
	}
	return nil
}

// Pop removes and returns the next scheduled job from the queue.
func (jq *jobQueue) Pop() (quartz.ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Pop")
	job, err := jq.findHead()
	if err == nil {
		if err = os.Remove(fmt.Sprintf("%s/%d", dataFolder, job.NextRunTime())); err != nil {
			jq.logger.Error("Failed to delete job", "error", err)
			return nil, err
		}
		return job, nil
	}
	jq.logger.Error("Failed to find job", "error", err)
	return nil, err
}

// Head returns the first scheduled job without removing it from the queue.
func (jq *jobQueue) Head() (quartz.ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Head")
	job, err := jq.findHead()
	if err != nil {
		jq.logger.Error("Failed to find job", "error", err)
	}
	return job, err
}

func (jq *jobQueue) findHead() (quartz.ScheduledJob, error) {
	fileInfo, err := os.ReadDir(dataFolder)
	if err != nil {
		return nil, err
	}
	var lastUpdate int64 = math.MaxInt64
	for _, file := range fileInfo {
		if !file.IsDir() {
			nextTime, err := strconv.ParseInt(file.Name(), 10, 64)
			if err == nil && nextTime < lastUpdate {
				lastUpdate = nextTime
			}
		}
	}
	if lastUpdate == math.MaxInt64 {
		return nil, quartz.ErrJobNotFound
	}
	data, err := os.ReadFile(fmt.Sprintf("%s/%d", dataFolder, lastUpdate))
	if err != nil {
		return nil, err
	}
	job, err := unmarshal(data, jq.logger)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// Get returns the scheduled job with the specified key without removing it
// from the queue.
func (jq *jobQueue) Get(jobKey *quartz.JobKey) (quartz.ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Get")
	fileInfo, err := os.ReadDir(dataFolder)
	if err != nil {
		return nil, err
	}
	for _, file := range fileInfo {
		if !file.IsDir() {
			data, err := os.ReadFile(fmt.Sprintf("%s/%s", dataFolder, file.Name()))
			if err == nil {
				job, err := unmarshal(data, jq.logger)
				if err == nil {
					if jobKey.Equals(job.JobDetail().JobKey()) {
						return job, nil
					}
				}
			}
		}
	}
	return nil, quartz.ErrJobNotFound
}

// Remove removes and returns the scheduled job with the specified key.
func (jq *jobQueue) Remove(jobKey *quartz.JobKey) (quartz.ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Remove")
	fileInfo, err := os.ReadDir(dataFolder)
	if err != nil {
		return nil, err
	}
	for _, file := range fileInfo {
		if !file.IsDir() {
			path := fmt.Sprintf("%s/%s", dataFolder, file.Name())
			data, err := os.ReadFile(path)
			if err == nil {
				job, err := unmarshal(data, jq.logger)
				if err == nil {
					if jobKey.Equals(job.JobDetail().JobKey()) {
						if err = os.Remove(path); err == nil {
							return job, nil
						}
					}
				}
			}
		}
	}
	return nil, quartz.ErrJobNotFound
}

// ScheduledJobs returns the slice of all scheduled jobs in the queue.
func (jq *jobQueue) ScheduledJobs(
	matchers []quartz.Matcher[quartz.ScheduledJob],
) ([]quartz.ScheduledJob, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("ScheduledJobs")
	var jobs []quartz.ScheduledJob
	fileInfo, err := os.ReadDir(dataFolder)
	if err != nil {
		return nil, err
	}
	for _, file := range fileInfo {
		if !file.IsDir() {
			data, err := os.ReadFile(fmt.Sprintf("%s/%s", dataFolder, file.Name()))
			if err == nil {
				job, err := unmarshal(data, jq.logger)
				if err == nil && isMatch(job, matchers) {
					jobs = append(jobs, job)
				}
			}
		}
	}
	return jobs, nil
}

func isMatch(job quartz.ScheduledJob, matchers []quartz.Matcher[quartz.ScheduledJob]) bool {
	for _, matcher := range matchers {
		// require all matchers to match the job
		if !matcher.IsMatch(job) {
			return false
		}
	}
	return true
}

// Size returns the size of the job queue.
func (jq *jobQueue) Size() (int, error) {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Size")
	files, err := os.ReadDir(dataFolder)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// Clear clears the job queue.
func (jq *jobQueue) Clear() error {
	jq.mtx.Lock()
	defer jq.mtx.Unlock()

	jq.logger.Trace("Clear")
	return os.RemoveAll(dataFolder)
}
