package eventing

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_outbox_repository_moq_test.go . OutboxRepository
type OutboxRepository interface {
	ReserveForPublishExecutionRequested(ctx context.Context, max_size int) ([]domain.ExecutionId, error)
	MarkAsPublishedExecutionRequested(ctx context.Context, execution_ids []domain.ExecutionId) error
}

//go:generate moq -out gen_outbox_store_moq_test.go . OutboxStore
type OutboxStore interface {
	Outbox() OutboxRepository
	WithinOutboxTx(ctx context.Context, fn func(ctx context.Context, outbox OutboxRepository) error) error
}
