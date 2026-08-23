package eventing

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_execution_publisher_moq_test.go . ExecutionPublisher
type ExecutionPublisher interface {
	Publish(ctx context.Context, executionId domain.ExecutionId) error
}
