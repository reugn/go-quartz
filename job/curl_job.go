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

// CurlJob represents a job that can be used to schedule HTTP requests.
// It implements the [quartz.Job] interface.
type CurlJob struct {
	mtx        sync.Mutex
	httpClient HTTPHandler
	request    *http.Request
	response   *http.Response
	jobStatus  Status

	once        sync.Once
	description string
	callback    func(context.Context, *CurlJob)
}

var _ quartz.Job = (*CurlJob)(nil)

// HTTPHandler sends an HTTP request and returns an HTTP response, following
// policy (such as redirects, cookies, auth) as configured on the implementing
// HTTP client.
type HTTPHandler interface {
	Do(req *http.Request) (*http.Response, error)
}

// CurlJobOptions represents optional parameters for constructing a [CurlJob].
type CurlJobOptions struct {
	HTTPClient HTTPHandler
	Callback   func(context.Context, *CurlJob)
}

// NewCurlJob returns a new [CurlJob] using the default HTTP client.
func NewCurlJob(request *http.Request) *CurlJob {
	return NewCurlJobWithOptions(request, CurlJobOptions{HTTPClient: http.DefaultClient})
}

// NewCurlJobWithOptions returns a new [CurlJob] configured with [CurlJobOptions].
func NewCurlJobWithOptions(request *http.Request, opts CurlJobOptions) *CurlJob {
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	return &CurlJob{
		httpClient: opts.HTTPClient,
		request:    request,
		jobStatus:  StatusNA,
		callback:   opts.Callback,
	}
}

// Description returns the description of the CurlJob.
func (cu *CurlJob) Description() string {
	cu.once.Do(func() {
		cu.description = formatRequest(cu.request)
	})
	return fmt.Sprintf("CurlJob%s%s", quartz.Sep, cu.description)
}

// DumpResponse returns the response of the job in its HTTP/1.x wire
// representation.
// If body is true, DumpResponse also returns the body.
func (cu *CurlJob) DumpResponse(body bool) ([]byte, error) {
	cu.mtx.Lock()
	defer cu.mtx.Unlock()
	if cu.response != nil {
		return httputil.DumpResponse(cu.response, body)
	}
	return nil, errors.New("response is nil")
}

// JobStatus returns the status of the CurlJob.
func (cu *CurlJob) JobStatus() Status {
	cu.mtx.Lock()
	defer cu.mtx.Unlock()
	return cu.jobStatus
}

func formatRequest(r *http.Request) string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "%v %v %v", r.Method, r.URL, r.Proto)
	for name, headers := range r.Header {
		for _, h := range headers {
			_, _ = fmt.Fprintf(&sb, "\n%v: %v", name, h)
		}
	}
	if r.ContentLength > 0 {
		_, _ = fmt.Fprintf(&sb, "\nContent Length: %d", r.ContentLength)
	}
	return sb.String()
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (cu *CurlJob) Execute(ctx context.Context) error {
	cu.mtx.Lock()
	cu.request = cu.request.WithContext(ctx)
	var err error
	cu.response, err = cu.httpClient.Do(cu.request)

	// update job status based on HTTP response code
	if cu.response != nil && cu.response.StatusCode >= http.StatusOK &&
		cu.response.StatusCode < http.StatusBadRequest {
		cu.jobStatus = StatusOK
	} else {
		cu.jobStatus = StatusFailure
	}
	cu.mtx.Unlock()

	if cu.callback != nil {
		cu.callback(ctx, cu)
	}
	return err
}
