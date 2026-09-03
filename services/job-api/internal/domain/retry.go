package domain

import (
	"time"
)

func shouldRetry(status ExecutionStatus, attempt int32, spec JobSpec, result *ExecutionResult) bool {
	if status != EXECUTION_STATUS_FAILED {
		return false
	}

	if attempt >= spec.RetryPolicy.MaxAttempts {
		return false
	}

	if result == nil {
		return false
	}

	switch spec.Type {
	case JOB_TYPE_HTTP_REQUEST:
		return shouldRetryHTTPJob(spec.HttpJobConfig(), result.HttpResult())
	default:
		return false
	}
}

func shouldRetryHTTPJob(cfg *HttpJobConfig, result *HttpExecutionResult) bool {
	if cfg == nil || result == nil {
		return false
	}

	if result.Error != nil {
		if cfg.RetryOnTimeout && result.Error.Type == HTTP_ERROR_TYPE_TIMEOUT {
			return true
		}
		if cfg.RetryOnNetworkError && result.Error.Type == HTTP_ERROR_TYPE_NETWORK {
			return true
		}
		return false
	}

	return cfg.RetryStatusCodes.Contains(result.StatusCode)
}

type retryDelayFunc func(policy RetryPolicy, attempt int32) time.Duration

type retryDelayStrategy struct {
	backoffType BackoffType
	delayFn     retryDelayFunc
}

var retryDelayStrategies = []retryDelayStrategy{
	{backoffType: BACKOFF_TYPE_FIXED, delayFn: fixedRetryDelay},
}

func computeRetryDelay(policy RetryPolicy, attempt int32) time.Duration {
	for _, strategy := range retryDelayStrategies {
		if strategy.backoffType == policy.BackoffType {
			return strategy.delayFn(policy, attempt)
		}
	}
	// fallback
	return fixedRetryDelay(policy, attempt)
}

func fixedRetryDelay(policy RetryPolicy, _ int32) time.Duration {
	return policy.InitialDelay
}
