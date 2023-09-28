package main

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/reugn/go-quartz/quartz"
)

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
	return quartz.HashCode(rsj.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
//func (rsj *ResultedShellJob) Execute(ctx context.Context) {
//	ch := ctx.Value("ch").(chan int)
//	fmt.Println("Executing " + rsj.Description())
//}

// NewShellJob returns a new ShellJob.
func NewShellJob(cmd string) *ResultedShellJob {
	return &ResultedShellJob{
		Cmd:       cmd,
		Result:    "",
		JobStatus: NA,
	}
}

// Description returns the description of the ShellJob.
//func (sh *ResultedShellJob) Description() string {
//	return fmt.Sprintf("ShellJob: %s", sh.Cmd)
//}

var (
	shellOnce = sync.Once{}
	shellPath = "bash"
)

func (rsj *ResultedShellJob) getShell() string {
	shellOnce.Do(func() {
		_, err := exec.LookPath("/bin/bash")
		// if not found bash binary, use `sh`.
		if err != nil {
			shellPath = "sh"
		}
	})
	return shellPath
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
