package quartz_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestFunctionJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var n int32 = 2
	funcJob1 := quartz.NewFunctionJob(func(_ context.Context) (string, error) {
		atomic.AddInt32(&n, 2)
		return "fired1", nil
	})

	funcJob2 := quartz.NewFunctionJob(func(_ context.Context) (int, error) {
		atomic.AddInt32(&n, 2)
		return 42, nil
	})

	sched := quartz.NewStdScheduler()
	sched.Start(ctx)
	sched.ScheduleJob(ctx, quartz.NewJobDetail(funcJob1, quartz.NewJobKey("funcJob1")),
		quartz.NewRunOnceTrigger(time.Millisecond*300))
	sched.ScheduleJob(ctx, quartz.NewJobDetail(funcJob2, quartz.NewJobKey("funcJob2")),
		quartz.NewRunOnceTrigger(time.Millisecond*800))
	time.Sleep(time.Second)
	_ = sched.Clear()
	sched.Stop()

	assertEqual(t, funcJob1.JobStatus(), quartz.OK)
	assertNotEqual(t, funcJob1.Result(), nil)
	assertEqual(t, *funcJob1.Result(), "fired1")

	assertEqual(t, funcJob2.JobStatus(), quartz.OK)
	assertNotEqual(t, funcJob2.Result(), nil)
	assertEqual(t, *funcJob2.Result(), 42)

	assertEqual(t, int(atomic.LoadInt32(&n)), 6)
}

func TestNewFunctionJobWithDesc(t *testing.T) {
	jobDesc := "test job"

	funcJob1 := quartz.NewFunctionJobWithDesc(jobDesc, func(_ context.Context) (string, error) {
		return "fired1", nil
	})

	funcJob2 := quartz.NewFunctionJobWithDesc(jobDesc, func(_ context.Context) (string, error) {
		return "fired2", nil
	})

	assertEqual(t, funcJob1.Description(), jobDesc)
	assertEqual(t, funcJob2.Description(), jobDesc)
}

func TestFunctionJobRespectsContext(t *testing.T) {
	var n int
	funcJob2 := quartz.NewFunctionJob(func(ctx context.Context) (bool, error) {
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
	go func() { defer close(sig); funcJob2.Execute(ctx) }()

	if n != 0 {
		t.Fatal("job should not have run yet")
	}
	cancel()
	<-sig

	if n != -1 {
		t.Fatal("job side effect should have reflected cancelation:", n)
	}
	if !errors.Is(funcJob2.Error(), context.Canceled) {
		t.Fatal("unexpected error function", funcJob2.Error())
	}
	if funcJob2.Result() != nil {
		t.Fatal("errored jobs should not return values")
	}
}
