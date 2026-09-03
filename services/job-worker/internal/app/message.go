// Package app contains the job-worker application types and use cases.
package app

import (
	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"
	"go.opentelemetry.io/otel/trace"
)

// QueuedItem is an item waiting to be executed.
type QueuedItem struct {
	ParentSpan trace.SpanContext
	Item       WorkItem
}

// WorkItem contains an execution to run.
type WorkItem struct {
	Execution *jobv1.ClaimedExecution
}

// QueuedResult is a completed result waiting to be reported.
type QueuedResult struct {
	ParentSpan trace.SpanContext
	Result     WorkResult
}

// WorkResult contains the outcome of an execution.
type WorkResult struct {
	ExecutionID string
	LeaseToken  string
	Status      jobv1.ExecutionStatus
	Result      *jobv1.ExecutionResult
}
