package worker

import (
	"bytes"
	"context"
	"io"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"net"
	"net/http"

	"github.com/cockroachdb/errors"
)

type httpExecutor struct {
	client http.Client
}

func (e *httpExecutor) Exec(ctx context.Context, item app.WorkItem) (app.WorkResult, error) {
	job_config := item.Execution.GetJobSpec().GetHttp()
	if job_config == nil {
		return app.WorkResult{}, errors.New("failed to get http")
	}

	method, err := httpMethodFromProto(job_config.GetMethod())
	if err != nil {
		return app.WorkResult{}, errors.Wrap(err, "failed to httpMethodFromProto")
	}
	var body io.Reader
	if len(job_config.GetBody()) > 0 && methodAllowsBody(job_config.GetMethod()) {
		body = bytes.NewReader(job_config.GetBody())
	}
	req, err := http.NewRequestWithContext(ctx, method, job_config.GetUrl(), body)
	if err != nil {
		return app.WorkResult{}, errors.Wrap(err, "failed to new http request")
	}
	for _, h := range job_config.GetHeaders() {
		req.Header.Add(h.GetName(), h.GetValue())
	}
	res, err := e.client.Do(req)
	if err != nil {
		err_type := classifyHttpError(err)
		return app.WorkResult{
			ExecutionID: item.Execution.ExecutionId,
			LeaseToken:  item.Execution.LeaseToken,
			Result: &jobv1.ExecutionResult{
				Result: &jobv1.ExecutionResult_Http{
					Http: &jobv1.HttpExecutionResult{
						Error: &jobv1.HttpError{
							Type:    err_type,
							Message: err.Error(),
						},
					},
				},
			},
		}, nil
	}
	defer res.Body.Close()
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

func classifyHttpError(err error) jobv1.HttpErrorType {
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

var errRedirectsDisabled = errors.New("redirects disabled")

func newHttpExecutor(item app.WorkItem) *httpExecutor {
	job_config := item.Execution.GetJobSpec().GetHttp()
	return &httpExecutor{
		client: http.Client{
			Timeout: job_config.GetTimeout().AsDuration(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !job_config.GetFollowRedirects() {
					return errRedirectsDisabled
				}
				return nil
			},
		},
	}
}
