package quartz

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

// Job represents an interface to be implemented by structs which represent a 'job'
// to be performed.
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated with this job fires.
	Execute()

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
	return fmt.Sprintf("ShellJob: %s.", sh.Cmd)
}

// Key returns the unique ShellJob key.
func (sh *ShellJob) Key() int {
	return HashCode(sh.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (sh *ShellJob) Execute() {
	out, err := exec.Command("sh", "-c", sh.Cmd).Output()
	if err != nil {
		sh.JobStatus = FAILURE
		sh.Result = err.Error()
		return
	}

	sh.JobStatus = OK
	sh.Result = string(out)
}

// CurlJob represents a cURL command Job, implements the quartz.Job interface.
// cURL is a command-line tool for getting or sending data including files using URL syntax.
type CurlJob struct {
	RequestMethod string
	URL           string
	Body          string
	Headers       map[string]string
	Response      string
	StatusCode    int
	JobStatus     JobStatus
	request       *http.Request
}

// NewCurlJob returns a new CurlJob.
func NewCurlJob(
	method string,
	url string,
	body string,
	headers map[string]string,
) (*CurlJob, error) {
	_body := bytes.NewBuffer([]byte(body))
	req, err := http.NewRequest(method, url, _body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return &CurlJob{
		RequestMethod: method,
		URL:           url,
		Body:          body,
		Headers:       headers,
		Response:      "",
		StatusCode:    -1,
		JobStatus:     NA,
		request:       req,
	}, nil
}

// Description returns the description of the CurlJob.
func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob: %s %s %s", cu.RequestMethod, cu.URL, cu.Body)
}

// Key returns the unique CurlJob key.
func (cu *CurlJob) Key() int {
	return HashCode(cu.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (cu *CurlJob) Execute() {
	client := &http.Client{}
	resp, err := client.Do(cu.request)
	if err != nil {
		cu.JobStatus = FAILURE
		cu.StatusCode = -1
		cu.Response = err.Error()
		return
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		cu.JobStatus = OK
	} else {
		cu.JobStatus = FAILURE
	}

	cu.StatusCode = resp.StatusCode
	cu.Response = string(body)
}
