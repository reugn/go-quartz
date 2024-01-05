package mock

import (
	"net/http"

	"github.com/reugn/go-quartz/job"
)

type HTTPHandlerMock struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m HTTPHandlerMock) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

var (
	HTTPHandlerOk  job.HTTPHandler
	HTTPHandlerErr job.HTTPHandler
)

func init() {
	HTTPHandlerMockOk := struct{ HTTPHandlerMock }{}
	HTTPHandlerMockOk.DoFunc = func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Request:    request,
		}, nil
	}
	HTTPHandlerOk = job.HTTPHandler(HTTPHandlerMockOk)

	HTTPHandlerMockErr := struct{ HTTPHandlerMock }{}
	HTTPHandlerMockErr.DoFunc = func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Request:    request,
		}, nil
	}
	HTTPHandlerErr = job.HTTPHandler(HTTPHandlerMockErr)
}
