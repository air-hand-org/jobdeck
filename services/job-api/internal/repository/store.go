package repository

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	q    *jobdb.Queries
}

func NewStore(p *pgxpool.Pool) *Store {
	return &Store{pool: p, q: jobdb.New(p)}
}

var _ app.JobStore = (*Store)(nil)
var _ app.ExecutionStore = (*Store)(nil)
var _ app.ExecutionQueueStore = (*Store)(nil)

func (s *Store) Jobs() app.JobRepository {
	return newJobRepository(s.q)
}

func (s *Store) WithinJobTx(ctx context.Context, fn func(ctx context.Context, jobs app.JobRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	jobs := newJobRepository(s.q.WithTx(tx))
	if err := fn(ctx, jobs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Executions() app.ExecutionRepository {
	return newExecutionRepository(s.q)
}

func (s *Store) WithinExecutionTx(ctx context.Context, fn func(ctx context.Context, execs app.ExecutionRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	execs := newExecutionRepository(s.q.WithTx(tx))
	if err := fn(ctx, execs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Queues() app.ExecutionQueueRepository {
	return newExecutionQueueRepository(s.q)
}

func (s *Store) WithinExecutionQueueTx(ctx context.Context, fn func(ctx context.Context, queues app.ExecutionQueueRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queues := newExecutionQueueRepository(s.q.WithTx(tx))
	if err := fn(ctx, queues); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
