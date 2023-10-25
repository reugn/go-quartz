package quartz

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/reugn/go-quartz/quartz/logger"
)

// Job represents an interface to be implemented by structs which
// represent a 'job' to be performed.
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated
	// with this job fires.
	Execute(context.Context)

	// Description returns the description of the Job.
	Description() string

	// Key returns the unique key for the Job.
	Key() int
}

// JobStatus represents a Job status.
type JobStatus int8

const (
	// NA is the initial Job status.
	NA JobStatus = iota

	// OK indicates the Job was completed successfully.
	OK

	// FAILURE indicates the Job failed.
	FAILURE
)

// ShellJob represents a shell command Job, implements the quartz.Job interface.
// Be aware of runtime.GOOS when sending shell commands for execution.
type ShellJob struct {
	sync.Mutex
	cmd       string
	exitCode  int
	stdout    string
	stderr    string
	jobStatus JobStatus
	callback  func(context.Context, *ShellJob)
}

// NewShellJob returns a new ShellJob for the given command.
func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: NA,
	}
}

// NewShellJobWithCallback returns a new ShellJob with the given callback function.
func NewShellJobWithCallback(cmd string, f func(context.Context, *ShellJob)) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: NA,
		callback:  f,
	}
}

// Description returns the description of the ShellJob.
func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob: %s", sh.cmd)
}

// Key returns the unique ShellJob key.
func (sh *ShellJob) Key() int {
	return HashCode(sh.Description())
}

var (
	shellOnce = sync.Once{}
	shellPath = "bash"
)

func getShell() string {
	shellOnce.Do(func() {
		_, err := exec.LookPath("/bin/bash")
		// if not found bash binary, use `sh`.
		if err != nil {
			shellPath = "sh"
		}
	})
	return shellPath
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (sh *ShellJob) Execute(ctx context.Context) {
	shell := getShell()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, shell, "-c", sh.cmd)
	cmd.Stdout = io.Writer(&stdout)
	cmd.Stderr = io.Writer(&stderr)

	err := cmd.Run()

	sh.Lock()
	sh.stdout = stdout.String()
	sh.stderr = stderr.String()
	sh.exitCode = cmd.ProcessState.ExitCode()

	if err != nil {
		sh.jobStatus = FAILURE
	} else {
		sh.jobStatus = OK
	}

	if sh.callback != nil {
		sh.callback(ctx, sh)
	}
	sh.Unlock()
}

// ExitCode returns the exit code of the ShellJob.
func (sh *ShellJob) ExitCode() int {
	sh.Lock()
	defer sh.Unlock()
	return sh.exitCode
}

// Stdout returns the captured stdout output of the ShellJob.
func (sh *ShellJob) Stdout() string {
	sh.Lock()
	defer sh.Unlock()
	return sh.stdout
}

// Stderr returns the captured stderr output of the ShellJob.
func (sh *ShellJob) Stderr() string {
	sh.Lock()
	defer sh.Unlock()
	return sh.stderr
}

// JobStatus returns the status of the ShellJob.
func (sh *ShellJob) JobStatus() JobStatus {
	sh.Lock()
	defer sh.Unlock()
	return sh.jobStatus
}

// CurlJob represents a cURL command Job, implements the quartz.Job interface.
// cURL is a command-line tool for getting or sending data including files
// using URL syntax.
type CurlJob struct {
	sync.Mutex
	httpClient  HTTPHandler
	request     *http.Request
	response    *http.Response
	jobStatus   JobStatus
	description string
	callback    func(context.Context, *CurlJob)
}

// HTTPHandler sends an HTTP request and returns an HTTP response,
// following policy (such as redirects, cookies, auth) as configured
// on the implementing HTTP client.
type HTTPHandler interface {
	Do(req *http.Request) (*http.Response, error)
}

// CurlJobOptions represents optional parameters for constructing a CurlJob.
type CurlJobOptions struct {
	HTTPClient HTTPHandler
	Callback   func(context.Context, *CurlJob)
}

// NewCurlJob returns a new CurlJob using the default HTTP client.
func NewCurlJob(request *http.Request) *CurlJob {
	return NewCurlJobWithOptions(request, CurlJobOptions{HTTPClient: http.DefaultClient})
}

// NewCurlJobWithOptions returns a new CurlJob configured with CurlJobOptions.
func NewCurlJobWithOptions(request *http.Request, opts CurlJobOptions) *CurlJob {
	return &CurlJob{
		httpClient:  opts.HTTPClient,
		request:     request,
		jobStatus:   NA,
		description: formatRequest(request),
		callback:    opts.Callback,
	}
}

// Description returns the description of the CurlJob.
func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob:\n%s", cu.description)
}

// Key returns the unique CurlJob key.
func (cu *CurlJob) Key() int {
	return HashCode(cu.description)
}

// Response returns the response of the CurlJob.
func (cu *CurlJob) Response() *http.Response {
	cu.Lock()
	defer cu.Unlock()
	return cu.response
}

// JobStatus returns the status of the CurlJob.
func (cu *CurlJob) JobStatus() JobStatus {
	cu.Lock()
	defer cu.Unlock()
	return cu.jobStatus
}

func formatRequest(r *http.Request) string {
	var request []string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	for name, headers := range r.Header {
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}
	if r.ContentLength > 0 {
		request = append(request, fmt.Sprintf("Content Length: %d", r.ContentLength))
	}
	return strings.Join(request, "\n")
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (cu *CurlJob) Execute(ctx context.Context) {
	cu.Lock()
	defer cu.Unlock()

	cu.request = cu.request.WithContext(ctx)
	var err error
	cu.response, err = cu.httpClient.Do(cu.request)

	if err == nil && cu.response.StatusCode >= 200 && cu.response.StatusCode < 400 {
		cu.jobStatus = OK
	} else {
		cu.jobStatus = FAILURE
	}

	if cu.callback != nil {
		cu.callback(ctx, cu)
	}
}

type isolatedJob struct {
	Job
	// TODO: switch this to an atomic.Bool when upgrading to/past go1.19
	isRunning *atomic.Value
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (j *isolatedJob) Execute(ctx context.Context) {
	if wasRunning := j.isRunning.Swap(true); wasRunning != nil && wasRunning.(bool) {
		logger.Debugf("Executed job %d is running.", j.Job.Key())
		return
	}
	defer j.isRunning.Store(false)

	j.Job.Execute(ctx)
}

// NewIsolatedJob wraps a job object and ensures that only one
// instance of the job's Execute method can be called at a time.
func NewIsolatedJob(underlying Job) Job {
	return &isolatedJob{
		Job:       underlying,
		isRunning: &atomic.Value{},
	}
}
