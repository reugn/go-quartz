package quartz

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
)

// Job is the interface to be implemented by structs which represent a 'job'
// to be performed.
type Job interface {
	// Execute Called by the Scheduler when a Trigger fires that is associated with the Job.
	Execute()

	// Description returns a Job description.
	Description() string

	// Key returns a Job unique key.
	Key() int
}

// JobStatus represents a job status.
type JobStatus int8

const (
	// NA is the initial JobStatus.
	NA JobStatus = iota
	// OK represents a successful JobStatus.
	OK
	// FAILURE represents a failed JobStatus.
	FAILURE
)

// ShellJob is the shell command Job, implements the quartz.Job interface.
// Consider the runtime.GOOS when sending the shell command to execute.
type ShellJob struct {
	Cmd       string
	Result    string
	JobStatus JobStatus
}

// NewShellJob returns a new ShellJob.
func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{cmd, "", NA}
}

// Description returns a ShellJob description.
func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob: %s.", sh.Cmd)
}

// Key returns a ShellJob unique key.
func (sh *ShellJob) Key() int {
	return HashCode(sh.Description())
}

// Execute Called by the Scheduler when a Trigger fires that is associated with the Job.
func (sh *ShellJob) Execute() {
	out, err := exec.Command("sh", "-c", sh.Cmd).Output()
	if err != nil {
		sh.JobStatus = FAILURE
		sh.Result = err.Error()
		return
	}

	sh.JobStatus = OK
	sh.Result = string(out[:])
}

// CurlJob is the curl command Job, implements the quartz.Job interface.
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

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
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

// Description returns a CurlJob description.
func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob: %s %s %s", cu.RequestMethod, cu.URL, cu.Body)
}

// Key returns a CurlJob unique key.
func (cu *CurlJob) Key() int {
	return HashCode(cu.Description())
}

// Execute Called by the Scheduler when a Trigger fires that is associated with the Job.
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
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		cu.JobStatus = OK
	} else {
		cu.JobStatus = FAILURE
	}

	cu.StatusCode = resp.StatusCode
	cu.Response = string(body[:])
}
