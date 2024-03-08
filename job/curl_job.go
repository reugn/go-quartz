package job

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/reugn/go-quartz/quartz"
)

// CurlJob represents a cURL command Job, implements the quartz.Job interface.
// cURL is a command-line tool for getting or sending data including files
// using URL syntax.
type CurlJob struct {
	sync.Mutex
	httpClient  HTTPHandler
	request     *http.Request
	response    *http.Response
	jobStatus   Status
	description string
	callback    func(context.Context, *CurlJob)
}

var _ quartz.Job = (*CurlJob)(nil)

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
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	return &CurlJob{
		httpClient:  opts.HTTPClient,
		request:     request,
		jobStatus:   StatusNA,
		description: formatRequest(request),
		callback:    opts.Callback,
	}
}

// Description returns the description of the CurlJob.
func (cu *CurlJob) Description() string {
	return fmt.Sprintf("CurlJob%s%s", quartz.Sep, cu.description)
}

// DumpResponse returns the response of the job in its HTTP/1.x wire
// representation.
// If body is true, DumpResponse also returns the body.
func (cu *CurlJob) DumpResponse(body bool) ([]byte, error) {
	cu.Lock()
	defer cu.Unlock()
	if cu.response != nil {
		return httputil.DumpResponse(cu.response, body)
	}
	return nil, errors.New("response is nil")
}

// JobStatus returns the status of the CurlJob.
func (cu *CurlJob) JobStatus() Status {
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
func (cu *CurlJob) Execute(ctx context.Context) error {
	cu.Lock()
	cu.request = cu.request.WithContext(ctx)
	var err error
	cu.response, err = cu.httpClient.Do(cu.request)

	if err == nil && cu.response.StatusCode >= 200 && cu.response.StatusCode < 400 {
		cu.jobStatus = StatusOK
	} else {
		cu.jobStatus = StatusFailure
	}
	cu.Unlock()

	if cu.callback != nil {
		cu.callback(ctx, cu)
	}
	return nil
}
