package quartz_test

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
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

func TestCurlJobExecute(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		jobStatus  quartz.JobStatus
		statusCode int
		respBody   string
	}{
		{
			name: "Http 200 Success",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusOK)
				_, err := rw.Write([]byte(`Http 200 Response`))
				if err != nil {
					return
				}
			}),
			jobStatus:  quartz.OK,
			statusCode: http.StatusOK,
			respBody:   "Http 200 Response",
		},
		{
			name: "Http 302 Success",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusFound)
				_, err := rw.Write([]byte(`Http 302 Response`))
				if err != nil {
					return
				}
			}),
			jobStatus:  quartz.OK,
			statusCode: http.StatusFound,
			respBody:   "Http 302 Response",
		},
		{
			name: "Http 500 Failure",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
				_, err := rw.Write([]byte(`Http 500 Response`))
				if err != nil {
					return
				}
			}),
			jobStatus:  quartz.FAILURE,
			statusCode: http.StatusInternalServerError,
			respBody:   "Http 500 Response",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set mock client to http client
	mockClient := &http.Client{}
	quartz.SetHTTPClient(mockClient)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			// create a new CurlJob
			job, err := quartz.NewCurlJob(http.MethodGet, server.URL, "", nil)
			assertEqual(t, err, nil)

			job.Execute(ctx)

			assertEqual(t, job.JobStatus, tt.jobStatus)
			assertEqual(t, job.StatusCode, tt.statusCode)
			assertEqual(t, job.Response, tt.respBody)
		})
	}
}

func TestCurlJobExecuteWithCancelledContext(t *testing.T) {
	// create a new CurlJob
	job, err := quartz.NewCurlJob(http.MethodGet, "https://example.com", "", nil)
	assertEqual(t, err, nil)

	// The context is canceled to test the job execute going down the unhappy path of
	// the http.Client returning an error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	job.Execute(ctx)

	assertEqual(t, job.JobStatus, quartz.FAILURE)
	assertEqual(t, job.StatusCode, -1)
	assertEqual(t, job.Response, "Get \"https://example.com\": context canceled")
}
