package repository

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/eventing"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type outboxQuerier interface {
	ReserveForPublishOutboxExecutionRequested(ctx context.Context, maxSize int32) ([]uuid.UUID, error)
	MarkAsPublishedOutboxExecutionRequested(ctx context.Context, executionIDs []uuid.UUID) ([]uuid.UUID, error)
}

var _ outboxQuerier = (*jobdb.Queries)(nil)

type outboxRepository struct {
	q outboxQuerier
}

var _ eventing.OutboxRepository = (*outboxRepository)(nil)

func (r *outboxRepository) ReserveForPublishExecutionRequested(ctx context.Context, maxSize int) ([]domain.ExecutionId, error) {
	res, err := r.q.ReserveForPublishOutboxExecutionRequested(ctx, int32(maxSize))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *outboxRepository) MarkAsPublishedExecutionRequested(ctx context.Context, executionIDs []domain.ExecutionId) error {
	res, err := r.q.MarkAsPublishedOutboxExecutionRequested(ctx, executionIDs)
	if err != nil {
		return err
	}
	if len(res) != len(executionIDs) {
		return errors.New("failed to update status as published")
	}
	return nil
}

func newOutboxRepository(q outboxQuerier) *outboxRepository {
	return &outboxRepository{q: q}
}
