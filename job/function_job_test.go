package job_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func TestFunctionJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var n int32 = 2
	funcJob1 := job.NewFunctionJob(func(_ context.Context) (string, error) {
		atomic.AddInt32(&n, 2)
		return "fired1", nil
	})

	funcJob2 := job.NewFunctionJob(func(_ context.Context) (int, error) {
		atomic.AddInt32(&n, 2)
		return 42, nil
	})

	sched := quartz.NewStdScheduler()
	sched.Start(ctx)
	assert.IsNil(t, sched.ScheduleJob(quartz.NewJobDetail(funcJob1, quartz.NewJobKey("funcJob1")),
		quartz.NewRunOnceTrigger(time.Millisecond*300)))
	assert.IsNil(t, sched.ScheduleJob(quartz.NewJobDetail(funcJob2, quartz.NewJobKey("funcJob2")),
		quartz.NewRunOnceTrigger(time.Millisecond*800)))
	time.Sleep(time.Second)
	assert.IsNil(t, sched.Clear())
	sched.Stop()

	assert.Equal(t, funcJob1.JobStatus(), job.StatusOK)
	assert.NotEqual(t, funcJob1.Result(), nil)
	assert.Equal(t, *funcJob1.Result(), "fired1")

	assert.Equal(t, funcJob2.JobStatus(), job.StatusOK)
	assert.NotEqual(t, funcJob2.Result(), nil)
	assert.Equal(t, *funcJob2.Result(), 42)

	assert.Equal(t, int(atomic.LoadInt32(&n)), 6)
}

func TestNewFunctionJobWithDesc(t *testing.T) {
	jobDesc := "test job"

	funcJob1 := job.NewFunctionJobWithDesc(jobDesc, func(_ context.Context) (string, error) {
		return "fired1", nil
	})

	funcJob2 := job.NewFunctionJobWithDesc(jobDesc, func(_ context.Context) (string, error) {
		return "fired2", nil
	})

	assert.Equal(t, funcJob1.Description(), jobDesc)
	assert.Equal(t, funcJob2.Description(), jobDesc)
}

func TestFunctionJobRespectsContext(t *testing.T) {
	var n int
	funcJob2 := job.NewFunctionJob(func(ctx context.Context) (bool, error) {
		timer := time.NewTimer(time.Hour)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			n--
			return false, ctx.Err()
		case <-timer.C:
			n++
			return true, nil
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan struct{})
	go func() { defer close(sig); _ = funcJob2.Execute(ctx) }()

	if n != 0 {
		t.Fatal("job should not have run yet")
	}
	cancel()
	<-sig

	if n != -1 {
		t.Fatal("job side effect should have reflected cancelation:", n)
	}
	assert.ErrorIs(t, funcJob2.Error(), context.Canceled)
	assert.IsNil(t, funcJob2.Result())
}
