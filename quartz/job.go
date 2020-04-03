package quartz

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
)

type Job interface {
	Execute()
	Description() string
	Key() int
}

type JobStatus int8

const (
	NA JobStatus = iota
	OK
	FAILURE
)

// ShellJob executes shell command
// need to take runtime.GOOS into consideration
type ShellJob struct {
	Cmd       string
	Result    string
	JobStatus JobStatus
}

func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{cmd, "", NA}
}

func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob: %s", sh.Cmd)
}

func (sh *ShellJob) Key() int {
	return HashCode(sh.Description())
}

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

// CurlJob executes curl command
type CurlJob struct {
	RequestMethod string
	Url           string
	Body          string
	Headers       map[string]string
	Response      string
	StatusCode    int
	JobStatus     JobStatus
	request       *http.Request
}

func NewCurlJob(method string, url string, body string, headers map[string]string) (*CurlJob, error) {
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
	return &CurlJob{method, url, body, headers, "", -1, NA, req}, nil
}

func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob: %s %s %s", cu.RequestMethod, cu.Url, cu.Body)
}

func (cu *CurlJob) Key() int {
	return HashCode(cu.Description())
}

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
