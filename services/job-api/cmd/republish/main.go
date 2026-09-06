// Command republish republishes pending executions.
package main

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type republishQuerier interface {
	RepublishOutboxExecutionRequested(ctx context.Context) ([]uuid.UUID, error)
}

type republishRepository struct {
	q republishQuerier
}

func (r *republishRepository) Republish(ctx context.Context) ([]domain.ExecutionId, error) {
	return r.q.RepublishOutboxExecutionRequested(ctx)
}

type republishStore struct {
	pool *pgxpool.Pool
	q    *jobdb.Queries
}

func (s *republishStore) WithinTx(ctx context.Context, fn func(ctx context.Context, repo republishRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := republishRepository{q: s.q.WithTx(tx)}
	if err := fn(ctx, repo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func newStore(p *pgxpool.Pool) *republishStore {
	return &republishStore{pool: p, q: jobdb.New(p)}
}

func main() {
	connStr := os.Getenv("POSTGRESDB_URI")
	if connStr == "" {
		log.Fatal("POSTGRESDB_URI is required")
	}

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	pool, err := pgxpool.New(signalCtx, connStr)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer pool.Close()

	store := newStore(pool)

	var execIDList []domain.ExecutionId
	if err := store.WithinTx(signalCtx, func(ctx context.Context, repo republishRepository) error {
		execIDs, err := repo.Republish(ctx)
		if err != nil {
			return err
		}
		execIDList = execIDs
		return nil
	}); err != nil {
		log.Fatalf("failed to republish: %v", err)
	}
	slog.Info(
		"republish execution ids",
		"id", execIDList,
	)
}
