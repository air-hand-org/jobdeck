package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	migrations "jobdeck/services/job-api/db/job"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestOutboxRepositoryReserveForPublishExecutionRequested_OnlyReservesDueQueuedExecutions(t *testing.T) {
	ctx := context.Background()
	pool := newMigratedTestPool(t, ctx)

	dueID := insertOutboxExecution(t, ctx, pool, "QUEUED", time.Now().Add(-time.Minute))
	insertOutboxExecution(t, ctx, pool, "QUEUED", time.Now().Add(time.Minute))
	insertOutboxExecution(t, ctx, pool, "RUNNING", time.Now().Add(-time.Minute))

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback(ctx)) })

	repository := newOutboxRepository(jobdb.New(tx))
	got, err := repository.ReserveForPublishExecutionRequested(ctx, 10)

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{dueID}, got)
}

func newMigratedTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(
		ctx,
		"postgres:18",
		postgres.WithDatabase("jobdeck"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	uri, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("pgx", uri)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("public.goose_db_version")
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.UpContext(ctx, db, "migrations"))

	pool, err := pgxpool.New(ctx, uri)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func insertOutboxExecution(t *testing.T, ctx context.Context, pool *pgxpool.Pool, status string, nextRunAt time.Time) uuid.UUID {
	t.Helper()

	jobID := uuid.New()
	revisionID := uuid.New()
	executionID := uuid.New()

	_, err := pool.Exec(ctx, `
INSERT INTO job.job (job_id, owner_id, enabled, name)
VALUES ($1, $2, TRUE, 'outbox integration test')
`, jobID, uuid.New())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO job.job_revision (
  revision_id, job_id, revision_number, type, config,
  retry_max_attempts, retry_backoff_type, retry_initial_delay_ms, retry_max_delay_ms
)
VALUES ($1, $2, 1, 'HTTP_REQUEST', '{}'::jsonb, 1, 'FIXED', 0, 0)
`, revisionID, jobID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO job.execution (
  execution_id, job_id, job_revision_id, trigger_type, status, attempt, next_run_at
)
VALUES ($1, $2, $3, 'MANUAL', $4, 1, $5)
`, executionID, jobID, revisionID, status, nextRunAt)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO job.outbox_execution_requested (execution_id)
VALUES ($1)
`, executionID)
	require.NoError(t, err)

	return executionID
}
