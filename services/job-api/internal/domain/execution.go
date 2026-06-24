package domain

import (
	"time"

	"github.com/google/uuid"
)

type ExecutionStatus string

const (
	EXECUTION_STATUS_QUEUED    ExecutionStatus = "QUEUED"
	EXECUTION_STATUS_RUNNING   ExecutionStatus = "RUNNING"
	EXECUTION_STATUS_SUCCEEDED ExecutionStatus = "SUCCEEDED"
	EXECUTION_STATUS_FAILED    ExecutionStatus = "FAILED"
	EXECUTION_STATUS_TIMED_OUT ExecutionStatus = "TIMED_OUT"
	EXECUTION_STATUS_CANCELLED ExecutionStatus = "CANCELLED"
)

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

type ExecutionTrigger string

const (
	EXECUTION_TRIGGER_MANUAL ExecutionTrigger = "MANUAL"
)

func (t ExecutionTrigger) Valid() bool {
	switch t {
	case EXECUTION_TRIGGER_MANUAL:
		return true
	default:
		return false
	}
}

type ExecutionEvent struct {
	Status    ExecutionStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

type Result interface {
	Result()
}

type HttpExecutionResult struct {
	StatusCode int32 `json:"status_code"`
}

func (r *HttpExecutionResult) Result() {}

var _ Result = (*HttpExecutionResult)(nil)

type ExecutionResult struct {
	Result       Result `json:"result"`
	ErrorMessage string `json:"error_message"`
}

func (r ExecutionResult) HttpResult() *HttpExecutionResult {
	if result, ok := r.Result.(*HttpExecutionResult); ok {
		return result
	}
	return nil
}

type ExecutionId = uuid.UUID
type LeaseToken = uuid.UUID

type Execution struct {
	Id      ExecutionId `json:"id"`
	JobId   JobId       `json:"job_id"`
	JobSpec JobSpec     `json:"job_spec"`

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

type ClaimedExecution struct {
	Id    ExecutionId `json:"id"`
	JobId JobId       `json:"job_id"`

	Attempt        int32      `json:"attempt"`
	LeaseToken     LeaseToken `json:"lease_token"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at"`
	JobSpec        JobSpec    `json:"job_spec"`
}
