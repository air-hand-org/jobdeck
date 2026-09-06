// Package app contains the job-worker application types and use cases.
package app

import "context"

// Claimer claims executions and queues them for processing.
type Claimer interface {
	ClaimAndEnqueue(ctx context.Context, executionID string) error
}

// PushUsecase handles execution requests received by the push endpoint.
type PushUsecase struct {
	claimer Claimer
}

// ClaimAndEnqueue claims an execution and queues it for processing.
func (u *PushUsecase) ClaimAndEnqueue(ctx context.Context, executionID string) error {
	return u.claimer.ClaimAndEnqueue(ctx, executionID)
}

// NewPushUsecase creates a push use case backed by c.
func NewPushUsecase(c Claimer) *PushUsecase {
	return &PushUsecase{
		claimer: c,
	}
}
