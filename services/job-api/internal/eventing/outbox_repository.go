package eventing

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

// OutboxRepository persists pending events.
//
//go:generate moq -out gen_outbox_repository_moq_test.go . OutboxRepository
type OutboxRepository interface {
	ReserveForPublishExecutionRequested(ctx context.Context, maxSize int) ([]domain.ExecutionId, error)
	MarkAsPublishedExecutionRequested(ctx context.Context, executionIDs []domain.ExecutionId) error
}

// OutboxStore provides transactional outbox access.
//
//go:generate moq -out gen_outbox_store_moq_test.go . OutboxStore
type OutboxStore interface {
	Outbox() OutboxRepository
	WithinOutboxTx(ctx context.Context, fn func(ctx context.Context, outbox OutboxRepository) error) error
}
