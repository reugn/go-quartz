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

// Job represents an interface to be implemented by structs which represent a 'job'
// to be performed.
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated with this job fires.
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
	Cmd       string
	Result    string
	ExitCode  int
	Stdout    string
	Stderr    string
	JobStatus JobStatus
}

// NewShellJob returns a new ShellJob.
func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{
		Cmd:       cmd,
		Result:    "",
		JobStatus: NA,
	}
}

// Description returns the description of the ShellJob.
func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob: %s", sh.Cmd)
}

// Key returns the unique ShellJob key.
func (sh *ShellJob) Key() int {
	return HashCode(sh.Description())
}

var (
	shellOnce = sync.Once{}
	shellPath = "bash"
)

func (sh *ShellJob) getShell() string {
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
	shell := sh.getShell()

	var stdout, stderr, result bytes.Buffer
	cmd := exec.CommandContext(ctx, shell, "-c", sh.Cmd)
	cmd.Stdout = io.MultiWriter(&stdout, &result)
	cmd.Stderr = io.MultiWriter(&stderr, &result)

	err := cmd.Run()
	sh.Stdout = stdout.String()
	sh.Stderr = stderr.String()
	sh.ExitCode = cmd.ProcessState.ExitCode()

	if err != nil {
		sh.JobStatus = FAILURE
		sh.Result = err.Error()
		return
	}

	sh.JobStatus = OK
	sh.Result = result.String()
}

// CurlJob represents a cURL command Job, implements the quartz.Job interface.
// cURL is a command-line tool for getting or sending data including files using URL syntax.
type CurlJob struct {
	httpClient  HTTPHandler
	request     *http.Request
	Response    *http.Response
	JobStatus   JobStatus
	description string
}

// HTTPHandler sends an HTTP request and returns an HTTP response,
// following policy (such as redirects, cookies, auth) as configured
// on the implementing HTTP client.
type HTTPHandler interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewCurlJob returns a new CurlJob using the default HTTP client.
func NewCurlJob(request *http.Request) (*CurlJob, error) {
	return NewCurlJobWithHTTPClient(request, http.DefaultClient)
}

// NewCurlJobWithHTTPClient returns a new CurlJob using a custom HTTP client.
func NewCurlJobWithHTTPClient(request *http.Request, httpClient HTTPHandler) (*CurlJob, error) {
	return &CurlJob{
		httpClient:  httpClient,
		request:     request,
		JobStatus:   NA,
		description: formatRequest(request),
	}, nil
}

// Description returns the description of the CurlJob.
func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob:\n%s", cu.description)
}

// Key returns the unique CurlJob key.
func (cu *CurlJob) Key() int {
	return HashCode(cu.description)
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
	cu.request = cu.request.WithContext(ctx)
	var err error
	cu.Response, err = cu.httpClient.Do(cu.request)

	if err == nil && cu.Response.StatusCode >= 200 && cu.Response.StatusCode < 400 {
		cu.JobStatus = OK
	} else {
		cu.JobStatus = FAILURE
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

type JobResult struct {
	Result string
	Err    string
	Code   int
}

// PrintJob implements the quartz.Job interface.
type ResultedShellJob struct {
	ScriptPath  string
	AcquireFunc func(path string) string
	Desc        string
	Cmd         string
	Result      string
	ExitCode    int
	Stdout      string
	Stderr      string
	JobStatus   JobStatus
}

// Description returns the description of the PrintJob.
func (rsj *ResultedShellJob) Description() string {
	return rsj.Desc
}

// Key returns the unique PrintJob key.
func (rsj *ResultedShellJob) Key() int {
	return HashCode(rsj.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
//func (rsj *ResultedShellJob) Execute(ctx context.Context) {
//	ch := ctx.Value("ch").(chan int)
//	fmt.Println("Executing " + rsj.Description())
//}

type AcquireFunc func(path string) string

// NewShellJob returns a new ShellJob.
func NewResultedShellJob(scriptPath string, acquireFunc AcquireFunc) *ResultedShellJob {
	return &ResultedShellJob{
		ScriptPath:  scriptPath,
		AcquireFunc: acquireFunc,
		Result:      "",
		JobStatus:   NA,
	}
}

// Description returns the description of the ShellJob.
//func (sh *ResultedShellJob) Description() string {
//	return fmt.Sprintf("ShellJob: %s", sh.Cmd)
//}

var (
	resultedShellOnce = sync.Once{}
	resultedShellPath = "bash"
)

func (rsj *ResultedShellJob) getShell() string {
	resultedShellOnce.Do(func() {
		_, err := exec.LookPath("/bin/bash")
		// if not found bash binary, use `sh`.
		if err != nil {
			resultedShellPath = "sh"
		}
	})
	return resultedShellPath
}

func (rsj *ResultedShellJob) acquireScript() {
	rsj.Cmd = rsj.AcquireFunc(rsj.ScriptPath)
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (rsj *ResultedShellJob) Execute(ctx context.Context) {
	ch := ctx.Value("jobresult").(chan JobResult)

	rsj.acquireScript()
	shell := rsj.getShell()

	var stdout, stderr, result bytes.Buffer
	cmd := exec.CommandContext(ctx, shell, "-c", rsj.Cmd)
	cmd.Stdout = io.MultiWriter(&stdout, &result)
	cmd.Stderr = io.MultiWriter(&stderr, &result)

	err := cmd.Run()
	rsj.Stdout = stdout.String()
	rsj.Stderr = stderr.String()
	rsj.ExitCode = cmd.ProcessState.ExitCode()

	jobResult := JobResult{
		Result: stdout.String(),
		Err:    stderr.String(),
		Code:   cmd.ProcessState.ExitCode(),
	}

	ch <- jobResult

	if err != nil {
		rsj.JobStatus = FAILURE
		rsj.Result = err.Error()
		return
	}

	rsj.JobStatus = OK
	rsj.Result = result.String()
}
