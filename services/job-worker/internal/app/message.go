package app

import (
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"

	"go.opentelemetry.io/otel/trace"
)

type QueuedItem struct {
	ParentSpan trace.SpanContext
	Item       WorkItem
}

type WorkItem struct {
	Execution *jobv1.ClaimedExecution
}

type QueuedResult struct {
	ParentSpan trace.SpanContext
	Result     WorkResult
}

type WorkResult struct {
	ExecutionID string
	LeaseToken  string
	Status      jobv1.ExecutionStatus
	Result      *jobv1.ExecutionResult
}
