// Package domain contains the job API domain model.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus is the lifecycle status of an execution.
type ExecutionStatus string

const (
	// EXECUTION_STATUS_QUEUED and related constants represent execution lifecycle states.
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_QUEUED ExecutionStatus = "QUEUED"
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_RUNNING ExecutionStatus = "RUNNING"
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_SUCCEEDED ExecutionStatus = "SUCCEEDED"
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_FAILED ExecutionStatus = "FAILED"
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_TIMED_OUT ExecutionStatus = "TIMED_OUT" // running, but too long until reported
	//nolint:revive // Names mirror the protobuf enum constants.
	EXECUTION_STATUS_CANCELLED ExecutionStatus = "CANCELLED" // cancelled before running
)

// Valid reports whether the execution status is recognized.
func (s ExecutionStatus) Valid() bool {
	switch s {
	case EXECUTION_STATUS_QUEUED,
		EXECUTION_STATUS_RUNNING,
		EXECUTION_STATUS_SUCCEEDED,
		EXECUTION_STATUS_FAILED,
		EXECUTION_STATUS_TIMED_OUT,
		EXECUTION_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

// ExecutionTrigger identifies what initiated an execution.
type ExecutionTrigger string

const (
	// EXECUTION_TRIGGER_MANUAL identifies a manually initiated execution.
	//nolint:revive // Name mirrors the protobuf enum constant.
	EXECUTION_TRIGGER_MANUAL ExecutionTrigger = "MANUAL"
)

// Valid reports whether the execution trigger is recognized.
func (t ExecutionTrigger) Valid() bool {
	switch t {
	case EXECUTION_TRIGGER_MANUAL:
		return true
	default:
		return false
	}
}

// ExecutionEvent records a status transition.
type ExecutionEvent struct {
	Status    ExecutionStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// Result is the result payload of an execution.
type Result interface {
	Result()
}

// HttpErrorType identifies an HTTP execution error.
//
//nolint:revive // Existing domain names are retained to avoid a broad API rename.
type HttpErrorType string

const (
	// HTTP_ERROR_TYPE_TIMEOUT and related constants identify transport failures.
	//nolint:revive // Names mirror the externally represented error types.
	HTTP_ERROR_TYPE_TIMEOUT HttpErrorType = "TIMEOUT"
	//nolint:revive // Name mirrors the protobuf enum constant.
	HTTP_ERROR_TYPE_NETWORK HttpErrorType = "NETWORK"
)

// Valid reports whether the HTTP error type is supported.
func (t HttpErrorType) Valid() bool {
	switch t {
	case HTTP_ERROR_TYPE_TIMEOUT, HTTP_ERROR_TYPE_NETWORK:
		return true
	default:
		return false
	}
}

// HttpError contains details of an HTTP failure.
//
//nolint:revive // Existing domain names are retained to avoid a broad API rename.
type HttpError struct {
	Type    HttpErrorType `json:"type"`
	Message string        `json:"message"`
}

// HttpExecutionResult contains the result of an HTTP request.
//
//nolint:revive // Existing domain names are retained to avoid a broad API rename.
type HttpExecutionResult struct {
	StatusCode int32      `json:"status_code"`
	Error      *HttpError `json:"error"`
}

// Result marks HttpExecutionResult as an execution result.
func (r *HttpExecutionResult) Result() {}

var _ Result = (*HttpExecutionResult)(nil)

// ExecutionResult wraps a concrete execution result.
type ExecutionResult struct {
	Result Result `json:"result"`
}

// HttpResult returns the HTTP result, if present.
//
//nolint:revive // Existing domain method name is part of the current API.
func (r ExecutionResult) HttpResult() *HttpExecutionResult {
	if result, ok := r.Result.(*HttpExecutionResult); ok {
		return result
	}
	return nil
}

// ExecutionId is the UUID of an execution.
//
//nolint:revive // Existing internal API name is retained for compatibility.
type ExecutionId = uuid.UUID

// LeaseToken identifies an execution lease.
type LeaseToken = uuid.UUID

// Execution is a requested or completed job execution.
type Execution struct {
	//nolint:revive // JSON field names are persisted API names.
	Id ExecutionId `json:"id"`
	//nolint:revive // JSON field names are persisted API names.
	JobId   JobId   `json:"job_id"`
	JobSpec JobSpec `json:"job_spec"`

	Trigger ExecutionTrigger `json:"trigger"`

	Status ExecutionStatus  `json:"status"`
	Events []ExecutionEvent `json:"events"`

	Attempt     int32 `json:"attempt"`
	MaxAttempts int32 `json:"max_attempts"`

	LeaseToken     LeaseToken `json:"lease_token"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at"`
	NextRunAt      time.Time  `json:"next_run_at"`

	Result *ExecutionResult `json:"result"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DetermineStatus computes the next status from an execution result.
func (e Execution) DetermineStatus(result ExecutionResult) (ExecutionStatus, error) {
	return determineExecutionStatus(e.JobSpec, result)
}

// ShouldRetry reports whether the execution should be retried.
func (e Execution) ShouldRetry() bool {
	return shouldRetry(e.Status, e.Attempt, e.JobSpec, e.Result)
}

// NextRetryRunAt returns the next retry time.
func (e Execution) NextRetryRunAt(now time.Time) time.Time {
	return now.Add(computeRetryDelay(e.JobSpec.RetryPolicy, e.Attempt))
}

// ClaimedExecution is an execution claimed by a worker.
type ClaimedExecution struct {
	//nolint:revive // JSON field names are persisted API names.
	Id ExecutionId `json:"id"`
	//nolint:revive // JSON field names are persisted API names.
	JobId JobId `json:"job_id"`

	Attempt        int32      `json:"attempt"`
	LeaseToken     LeaseToken `json:"lease_token"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at"`
	JobSpec        JobSpec    `json:"job_spec"`
}
