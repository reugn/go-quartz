package quartz_test

import (
	"context"
	"errors"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestMultipleExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var n int64
	job := quartz.NewIsolatedJob(quartz.NewFunctionJob(func(ctx context.Context) (bool, error) {
		atomic.AddInt64(&n, 1)
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
				t.Error("should not have timed out")
			}
		case <-timer.C:
			t.Error("should not have reached timeout")
		}

		return false, ctx.Err()
	}))

	// start a bunch of threads that run jobs
	sig := make(chan struct{})
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			timer := time.NewTimer(0)
			defer timer.Stop()
			count := 0
			defer func() {
				if count == 0 {
					t.Error("should run at least once")
				}
			}()
			for {
				count++
				select {
				case <-timer.C:
					// sleep for a jittered amount of
					// time, less than 11ms
					job.Execute(ctx)
				case <-ctx.Done():
					return
				case <-sig:
					return
				}
				timer.Reset(1 + time.Duration(rand.Int63n(10))*time.Millisecond)
			}
		}()
	}

	// check very often that we've only run one job
	ticker := time.NewTicker(2 * time.Millisecond)
	for i := 0; i < 1000; i++ {
		select {
		case <-ticker.C:
			if atomic.LoadInt64(&n) != 1 {
				t.Error("only one job should run")
			}
		case <-ctx.Done():
			t.Error("should not have reached timeout")
			break
		}
	}

	// stop all of the adding threads without canceling
	// the context
	close(sig)
	if atomic.LoadInt64(&n) != 1 {
		t.Error("only one job should run")
	}
}
