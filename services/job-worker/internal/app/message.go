package app

import (
	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"
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
