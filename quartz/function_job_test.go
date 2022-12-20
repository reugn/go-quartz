package quartz_test

import (
	"context"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestFunctionJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var n = 2
	funcJob1 := quartz.NewFunctionJob(func(_ context.Context) (string, error) {
		n += 2
		return "fired1", nil
	})

	funcJob2 := quartz.NewFunctionJob(func(_ context.Context) (int, error) {
		n += 2
		return 42, nil
	})

	sched := quartz.NewStdScheduler()
	sched.Start(ctx)
	sched.ScheduleJob(funcJob1, quartz.NewRunOnceTrigger(time.Millisecond*300))
	sched.ScheduleJob(funcJob2, quartz.NewRunOnceTrigger(time.Millisecond*800))
	time.Sleep(time.Second)
	sched.Clear()
	sched.Stop()

	assertEqual(t, funcJob1.JobStatus, quartz.OK)
	assertNotEqual(t, funcJob1.Result, nil)
	assertEqual(t, *funcJob1.Result, "fired1")

	assertEqual(t, funcJob2.JobStatus, quartz.OK)
	assertNotEqual(t, funcJob2.Result, nil)
	assertEqual(t, *funcJob2.Result, 42)

	assertEqual(t, n, 6)
}
