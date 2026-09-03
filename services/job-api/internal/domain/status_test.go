package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineStatus_HttpExpectedStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		expected ExecutionStatus
		spec     JobSpec
		result   ExecutionResult
	}{
		{
			name:     "200 success",
			expected: EXECUTION_STATUS_SUCCEEDED,
			spec:     httpJobSpec(),
			result:   httpResultResult(200),
		},
		{
			name:     "404 failed",
			expected: EXECUTION_STATUS_FAILED,
			spec:     httpJobSpec(),
			result:   httpResultResult(404),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineExecutionStatus(tt.spec, tt.result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetermineStatus_HttpErrorAsFailed(t *testing.T) {
	tests := []struct {
		name     string
		expected ExecutionStatus
		spec     JobSpec
		result   ExecutionResult
	}{
		{
			name:     "http timeout must be failed, not timed out",
			expected: EXECUTION_STATUS_FAILED,
			spec:     httpJobSpec(),
			result:   httpErrorResult(HTTP_ERROR_TYPE_TIMEOUT),
		},
		{
			name:     "network error must be failed",
			expected: EXECUTION_STATUS_FAILED,
			spec:     httpJobSpec(),
			result:   httpErrorResult(HTTP_ERROR_TYPE_NETWORK),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineExecutionStatus(tt.spec, tt.result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func httpJobSpec(opts ...func(*HttpJobConfig, *RetryPolicy)) JobSpec {
	cfg := &HttpJobConfig{
		ExpectedStatusCodes: StatusCodes{
			Codes: []int32{200},
		},
	}

	retry_policy := &RetryPolicy{
		MaxAttempts:  10,
		BackoffType:  BACKOFF_TYPE_FIXED,
		InitialDelay: 10 * time.Second,
		MaxDelay:     20 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg, retry_policy)
	}

	return JobSpec{
		Type:        JOB_TYPE_HTTP_REQUEST,
		Config:      cfg,
		RetryPolicy: *retry_policy,
	}
}

func httpResultResult(statusCode int32) ExecutionResult {
	return ExecutionResult{
		Result: &HttpExecutionResult{
			StatusCode: statusCode,
		},
	}
}

func httpErrorResult(error_type HttpErrorType) ExecutionResult {
	return ExecutionResult{
		Result: &HttpExecutionResult{
			Error: &HttpError{
				Type:    error_type,
				Message: string(error_type),
			},
		},
	}
}
