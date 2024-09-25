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

// ShellJob represents a shell command Job, implements the [quartz.Job] interface.
// The command will be executed using bash if available; otherwise, sh will be used.
// Consider the interpreter type and target environment when formulating commands
// for execution.
type ShellJob struct {
	mtx       sync.Mutex
	cmd       string
	exitCode  int
	stdout    string
	stderr    string
	jobStatus Status
	callback  func(context.Context, *ShellJob)
}

var _ quartz.Job = (*ShellJob)(nil)

// NewShellJob returns a new [ShellJob] for the given command.
func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: StatusNA,
	}
}

// NewShellJobWithCallback returns a new [ShellJob] with the given callback function.
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
		// if bash binary is not found, use `sh`.
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

	err := cmd.Run() // run the command

	sh.mtx.Lock()
	sh.stdout, sh.stderr = stdout.String(), stderr.String()
	sh.exitCode = cmd.ProcessState.ExitCode()

	if err != nil {
		sh.jobStatus = StatusFailure
	} else {
		sh.jobStatus = StatusOK
	}
	sh.mtx.Unlock()

	if sh.callback != nil {
		sh.callback(ctx, sh)
	}
	return err
}

// ExitCode returns the exit code of the ShellJob.
func (sh *ShellJob) ExitCode() int {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.exitCode
}

// Stdout returns the captured stdout output of the ShellJob.
func (sh *ShellJob) Stdout() string {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.stdout
}

// Stderr returns the captured stderr output of the ShellJob.
func (sh *ShellJob) Stderr() string {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.stderr
}

// JobStatus returns the status of the ShellJob.
func (sh *ShellJob) JobStatus() Status {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.jobStatus
}
