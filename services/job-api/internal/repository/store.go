package repository

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/eventing"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides repository implementations backed by PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
	q    *jobdb.Queries
}

// NewStore constructs a repository store.
func NewStore(p *pgxpool.Pool) *Store {
	return &Store{pool: p, q: jobdb.New(p)}
}

var _ app.JobStore = (*Store)(nil)
var _ app.ExecutionStore = (*Store)(nil)
var _ app.ExecutionQueueStore = (*Store)(nil)
var _ app.ActorStore = (*Store)(nil)
var _ eventing.OutboxStore = (*Store)(nil)

// Jobs returns the job repository.
func (s *Store) Jobs() app.JobRepository {
	return newJobRepository(s.q)
}

// WithinJobTx runs a job repository operation in a transaction.
func (s *Store) WithinJobTx(ctx context.Context, fn func(ctx context.Context, jobs app.JobRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	jobs := newJobRepository(s.q.WithTx(tx))
	if err := fn(ctx, jobs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Executions returns the execution repository.
func (s *Store) Executions() app.ExecutionRepository {
	return newExecutionRepository(s.q)
}

// WithinExecutionTx runs an execution repository operation in a transaction.
func (s *Store) WithinExecutionTx(ctx context.Context, fn func(ctx context.Context, execs app.ExecutionRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	execs := newExecutionRepository(s.q.WithTx(tx))
	if err := fn(ctx, execs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Queues returns the execution queue repository.
func (s *Store) Queues() app.ExecutionQueueRepository {
	return newExecutionQueueRepository(s.q)
}

// WithinExecutionQueueTx runs an execution queue repository operation in a transaction.
func (s *Store) WithinExecutionQueueTx(ctx context.Context, fn func(ctx context.Context, queues app.ExecutionQueueRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queues := newExecutionQueueRepository(s.q.WithTx(tx))
	if err := fn(ctx, queues); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Actors returns the actor repository.
func (s *Store) Actors() app.ActorRepository {
	return newActorRepository(s.q)
}

// WithinActorTx runs an actor repository operation in a transaction.
func (s *Store) WithinActorTx(ctx context.Context, fn func(ctx context.Context, actors app.ActorRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	actors := newActorRepository(s.q.WithTx(tx))
	if err := fn(ctx, actors); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Outbox returns the outbox repository.
func (s *Store) Outbox() eventing.OutboxRepository {
	return newOutboxRepository(s.q)
}

// WithinOutboxTx runs an outbox repository operation in a transaction.
func (s *Store) WithinOutboxTx(ctx context.Context, fn func(ctx context.Context, outbox eventing.OutboxRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	outbox := newOutboxRepository(s.q.WithTx(tx))
	if err := fn(ctx, outbox); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
