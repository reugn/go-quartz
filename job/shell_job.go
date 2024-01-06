package job

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/reugn/go-quartz/quartz"
)

// ShellJob represents a shell command Job, implements the quartz.Job interface.
// Be aware of runtime.GOOS when sending shell commands for execution.
type ShellJob struct {
	sync.Mutex
	cmd       string
	exitCode  int
	stdout    string
	stderr    string
	jobStatus Status
	callback  func(context.Context, *ShellJob)
}

var _ quartz.Job = (*ShellJob)(nil)

// NewShellJob returns a new ShellJob for the given command.
func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: StatusNA,
	}
}

// NewShellJobWithCallback returns a new ShellJob with the given callback function.
func NewShellJobWithCallback(cmd string, f func(context.Context, *ShellJob)) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: StatusNA,
		callback:  f,
	}
}

// Description returns the description of the ShellJob.
func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob%s%s", quartz.Sep, sh.cmd)
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
func (sh *ShellJob) Execute(ctx context.Context) error {
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
		sh.jobStatus = StatusFailure
	} else {
		sh.jobStatus = StatusOK
	}
	sh.Unlock()

	if sh.callback != nil {
		sh.callback(ctx, sh)
	}
	return nil
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
func (sh *ShellJob) JobStatus() Status {
	sh.Lock()
	defer sh.Unlock()
	return sh.jobStatus
}
