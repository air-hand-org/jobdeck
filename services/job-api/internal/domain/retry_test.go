package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldRetry_Complex(t *testing.T) {
	tests := []struct{
		name string
		expected bool
		attempt int32
		spec JobSpec
		result ExecutionResult
	}{
		{
			name: "retry on timeout",
			expected: true,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, true, false, StatusCodes{}),
			result: httpErrorResult(HTTP_ERROR_TYPE_TIMEOUT),
		},
		{
			name: "timeout occurred but not retry when timeout",
			expected: false,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, false, true, StatusCodes{}),
			result: httpErrorResult(HTTP_ERROR_TYPE_TIMEOUT),
		},
		{
			name: "retry on network error",
			expected: true,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, false, true, StatusCodes{}),
			result: httpErrorResult(HTTP_ERROR_TYPE_NETWORK),
		},
		{
			name: "network error occurred but not retry when network error",
			expected: false,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, true, false, StatusCodes{}),
			result: httpErrorResult(HTTP_ERROR_TYPE_NETWORK),
		},
		{
			name: "retry on status code",
			expected: true,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, false, false, StatusCodes{Codes: []int32{500}}),
			result: httpResultResult(500),
		},
		{
			name: "attempt reached max",
			expected: false,
			attempt: 2,
			spec: applyHttpCfgAndRetryPolicy(2, false, false, StatusCodes{Codes: []int32{500}}),
			result: httpResultResult(500),
		},
		{
			name: "retry 500, but not 503",
			expected: false,
			attempt: 1,
			spec: applyHttpCfgAndRetryPolicy(2, false, false, StatusCodes{Codes: []int32{500}}),
			result: httpResultResult(503),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetry(EXECUTION_STATUS_FAILED, tt.attempt, tt.spec, &tt.result)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShouldRetry_NotFailedAsNotRetriable(t *testing.T) {
	tests := []struct{
		name string
		status ExecutionStatus
	}{
		{
			name: "FAILED should retry",
			status: EXECUTION_STATUS_FAILED,
		},
		{
			name: "SUCCEEDED should not retry",
			status: EXECUTION_STATUS_SUCCEEDED,
		},
		{
			name: "TIMED_OUT should not retry",
			status: EXECUTION_STATUS_TIMED_OUT,
		},
		{
			name: "CANCELLED should not retry",
			status: EXECUTION_STATUS_CANCELLED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := 1
			spec := applyHttpCfgAndRetryPolicy(10, true, true, StatusCodes{Codes: []int32{500}})
			result := httpResultResult(500)
			got := shouldRetry(tt.status, int32(attempt), spec, &result)
			assert.Equal(t, tt.status == EXECUTION_STATUS_FAILED, got)
		})
	}
}

func TestComputeRetryDelay(t *testing.T) {
	tests := []struct{
		name string
		expected time.Duration
		attempt int32
		policy RetryPolicy
	}{
		{
			name: "fixed retry delay",
			expected: 1 * time.Second,
			attempt: 1,
			policy: RetryPolicy{
				BackoffType: BACKOFF_TYPE_FIXED,
				MaxAttempts: 10,
				InitialDelay: 1 * time.Second,
				MaxDelay: 10 * time.Second,
			},
		},
		{
			name: "fixed retry delay second time",
			expected: 1 * time.Second,
			attempt: 2,
			policy: RetryPolicy{
				BackoffType: BACKOFF_TYPE_FIXED,
				MaxAttempts: 10,
				InitialDelay: 1 * time.Second,
				MaxDelay: 10 * time.Second,
			},
		},
		{
			name: "unknown backoff type as fixed",
			expected: 2 * time.Second,
			attempt: 2,
			policy: RetryPolicy{
				MaxAttempts: 10,
				InitialDelay: 2 * time.Second,
				MaxDelay: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeRetryDelay(tt.policy, tt.attempt)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNextRetryRunAt(t *testing.T) {
	now := time.Date(2000, time.January, 1, 20, 11, 30, 0, time.UTC)
	tests := []struct{
		name string
		expected time.Time
		attempt int32
		spec JobSpec
	}{
		{
			name: "fixed next retry run at",
			expected: time.Date(2000, time.January, 1, 20, 11, 31, 0, time.UTC),
			attempt: 1,
			spec: applyRetryPolicy(BACKOFF_TYPE_FIXED, 3, 1 * time.Second, 10 * time.Second),
		},
		{
			name: "fixed next retry run at. delay 1 minute",
			expected: time.Date(2000, time.January, 1, 20, 12, 30, 0, time.UTC),
			attempt: 1,
			spec: applyRetryPolicy(BACKOFF_TYPE_FIXED, 3, 1 * time.Minute, 10 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := httpJobExecution(tt.spec, EXECUTION_STATUS_FAILED, tt.attempt, nil)
			got := exec.NextRetryRunAt(now)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func applyHttpCfgAndRetryPolicy(max_attempt int32, retry_on_timeout, retry_on_network_error bool, retry_status_codes StatusCodes) JobSpec {
	return httpJobSpec(func(cfg *HttpJobConfig, policy *RetryPolicy) {
		cfg.RetryOnTimeout = retry_on_timeout
		cfg.RetryOnNetworkError = retry_on_network_error
		cfg.RetryStatusCodes = retry_status_codes

		policy.MaxAttempts = max_attempt
	})
}

func applyRetryPolicy(backoff_type BackoffType, max_attempt int32, initial_delay time.Duration, max_delay time.Duration) JobSpec {
	return httpJobSpec(func(_ *HttpJobConfig, policy *RetryPolicy) {
		policy.BackoffType = backoff_type
		policy.MaxAttempts = max_attempt
		policy.InitialDelay = initial_delay
		policy.MaxDelay = max_delay
	})
}

func httpJobExecution(spec JobSpec, status ExecutionStatus, attempt int32, result *ExecutionResult) Execution {
	exec := Execution{
		Id: uuid.Max,
		JobId: uuid.Max,
		JobSpec: spec,
		Trigger: EXECUTION_TRIGGER_MANUAL,
		Status: status,
		Attempt: attempt,
		Result: result,
	}
	return exec
}
