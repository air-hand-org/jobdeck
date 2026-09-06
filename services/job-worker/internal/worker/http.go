// Package worker executes claimed HTTP jobs.
package worker

import (
	"bytes"
	"context"
	"io"
	"jobdeck/services/job-worker/internal/app"
	"net"
	"net/http"
	"net/url"
	"os"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"
	"github.com/cockroachdb/errors"
)

type httpExecutor struct {
	client http.Client
}

func (e *httpExecutor) Exec(ctx context.Context, item app.WorkItem) (app.WorkResult, error) {
	jobConfig := item.Execution.GetJobSpec().GetHttp()
	if jobConfig == nil {
		return app.WorkResult{}, errors.New("failed to get http")
	}

	method, err := httpMethodFromProto(jobConfig.GetMethod())
	if err != nil {
		return app.WorkResult{}, errors.Wrap(err, "failed to httpMethodFromProto")
	}
	var body io.Reader
	if len(jobConfig.GetBody()) > 0 && methodAllowsBody(jobConfig.GetMethod()) {
		body = bytes.NewReader(jobConfig.GetBody())
	}
	req, err := http.NewRequestWithContext(ctx, method, jobConfig.GetUrl(), body)
	if err != nil {
		return app.WorkResult{}, errors.Wrap(err, "failed to new http request")
	}
	for _, h := range jobConfig.GetHeaders() {
		req.Header.Add(h.GetName(), h.GetValue())
	}
	res, err := e.client.Do(req)
	if err != nil {
		errType := classifyHTTPError(err)
		return app.WorkResult{
			ExecutionID: item.Execution.ExecutionId,
			LeaseToken:  item.Execution.LeaseToken,
			Result: &jobv1.ExecutionResult{
				Result: &jobv1.ExecutionResult_Http{
					Http: &jobv1.HttpExecutionResult{
						Error: &jobv1.HttpError{
							Type:    errType,
							Message: err.Error(),
						},
					},
				},
			},
		}, nil
	}
	defer func() { _ = res.Body.Close() }()
	// b, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return app.WorkResult{}, err
	// }

	return app.WorkResult{

		ExecutionID: item.Execution.ExecutionId,
		LeaseToken:  item.Execution.LeaseToken,
		Result: &jobv1.ExecutionResult{
			Result: &jobv1.ExecutionResult_Http{
				Http: &jobv1.HttpExecutionResult{
					StatusCode: int32(res.StatusCode),
				},
			},
		},
	}, nil
}

func classifyHTTPError(err error) jobv1.HttpErrorType {
	if err == nil {
		return jobv1.HttpErrorType_HTTP_ERROR_TYPE_UNSPECIFIED
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return jobv1.HttpErrorType_HTTP_ERROR_TYPE_TIMEOUT
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return jobv1.HttpErrorType_HTTP_ERROR_TYPE_TIMEOUT
	}

	return jobv1.HttpErrorType_HTTP_ERROR_TYPE_NETWORK
}

func methodAllowsBody(m jobv1.HttpMethod) bool {
	switch m {
	case jobv1.HttpMethod_HTTP_METHOD_PUT,
		jobv1.HttpMethod_HTTP_METHOD_POST,
		jobv1.HttpMethod_HTTP_METHOD_PATCH:
		return true
	default:
		return false
	}
}

func httpMethodFromProto(m jobv1.HttpMethod) (string, error) {
	switch m {
	case jobv1.HttpMethod_HTTP_METHOD_GET:
		return "GET", nil
	case jobv1.HttpMethod_HTTP_METHOD_HEAD:
		return "HEAD", nil
	case jobv1.HttpMethod_HTTP_METHOD_POST:
		return "POST", nil
	case jobv1.HttpMethod_HTTP_METHOD_PUT:
		return "PUT", nil
	case jobv1.HttpMethod_HTTP_METHOD_PATCH:
		return "PATCH", nil
	case jobv1.HttpMethod_HTTP_METHOD_DELETE:
		return "DELETE", nil
	default:
		return "", errors.New("unsupported http method")
	}
}

var errProxySettingsNotFound = errors.New("WORKER_HTTP_PROXY env is not set")
var errRedirectsDisabled = errors.New("redirects disabled")

func newHTTPExecutor(item app.WorkItem) (*httpExecutor, error) {
	proxyURLStr := os.Getenv("WORKER_HTTP_PROXY")
	if proxyURLStr == "" {
		return nil, errProxySettingsNotFound
	}
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse proxy URL")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)

	jobConfig := item.Execution.GetJobSpec().GetHttp()
	return &httpExecutor{
		client: http.Client{
			Transport: transport,
			Timeout:   jobConfig.GetTimeout().AsDuration(),
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				if !jobConfig.GetFollowRedirects() {
					return errRedirectsDisabled
				}
				return nil
			},
		},
	}, nil
}
