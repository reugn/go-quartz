package quartz_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
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

type httpHandlerMock struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m httpHandlerMock) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

var worldtimeapiURL = "https://worldtimeapi.org/api/timezone/utc"

func TestCurlJob(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, worldtimeapiURL, nil)
	if err != nil {
		t.Error(err)
	}
	handlerOk := struct{ httpHandlerMock }{}
	handlerOk.doFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Request:    request,
		}, nil
	}
	handlerErr := struct{ httpHandlerMock }{}
	handlerErr.doFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Request:    request,
		}, nil
	}

	tests := []struct {
		name           string
		request        *http.Request
		opts           quartz.CurlJobOptions
		expectedStatus quartz.JobStatus
	}{
		{
			name:           "HTTP 200 OK",
			request:        request,
			opts:           quartz.CurlJobOptions{HTTPClient: handlerOk},
			expectedStatus: quartz.OK,
		},
		{
			name:           "HTTP 500 Internal Server Error",
			request:        request,
			opts:           quartz.CurlJobOptions{HTTPClient: handlerErr},
			expectedStatus: quartz.FAILURE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpJob := quartz.NewCurlJobWithOptions(tt.request, tt.opts)
			httpJob.Execute(context.Background())
			assertEqual(t, httpJob.JobStatus(), tt.expectedStatus)
		})
	}
}

func TestCurlJobDescription(t *testing.T) {
	postRequest, err := http.NewRequest(
		http.MethodPost,
		worldtimeapiURL,
		strings.NewReader("{\"a\":1}"),
	)
	if err != nil {
		t.Error(err)
	}
	postRequest.Header = http.Header{
		"Content-Type": {"application/json"},
	}
	getRequest, err := http.NewRequest(
		http.MethodGet,
		worldtimeapiURL,
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name                string
		request             *http.Request
		expectedDescription string
	}{
		{
			name:    "POST with headers and body",
			request: postRequest,
			expectedDescription: "CurlJob:\n" +
				fmt.Sprintf("POST %s HTTP/1.1\n", worldtimeapiURL) +
				"Content-Type: application/json\n" +
				"Content Length: 7",
		},
		{
			name:    "Get request",
			request: getRequest,
			expectedDescription: "CurlJob:\n" +
				fmt.Sprintf("GET %s HTTP/1.1", worldtimeapiURL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := quartz.CurlJobOptions{HTTPClient: http.DefaultClient}
			httpJob := quartz.NewCurlJobWithOptions(tt.request, opts)
			assertEqual(t, httpJob.Description(), tt.expectedDescription)
		})
	}
}

func TestShellJob_Execute(t *testing.T) {
	type args struct {
		Cmd      string
		ExitCode int
		Result   string
		Stdout   string
		Stderr   string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "test stdout",
			args: args{
				Cmd:      "echo -n ok",
				ExitCode: 0,
				Stdout:   "ok",
				Stderr:   "",
			},
		},
		{
			name: "test stderr",
			args: args{
				Cmd:      "echo -n err >&2",
				ExitCode: 0,
				Stdout:   "",
				Stderr:   "err",
			},
		},
		{
			name: "test combine",
			args: args{
				Cmd:      "echo -n ok && sleep 0.01 && echo -n err >&2",
				ExitCode: 0,
				Stdout:   "ok",
				Stderr:   "err",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := quartz.NewShellJob(tt.args.Cmd)
			sh.Execute(context.TODO())

			assertEqual(t, tt.args.ExitCode, sh.ExitCode())
			assertEqual(t, tt.args.Stderr, sh.Stderr())
			assertEqual(t, tt.args.Stdout, sh.Stdout())
		})
	}

	// invalid command
	stdoutShell := "invalid_command"
	sh := quartz.NewShellJob(stdoutShell)
	sh.Execute(context.Background())
	assertEqual(t, 127, sh.ExitCode())
	// the return value is different under different platforms.
}
