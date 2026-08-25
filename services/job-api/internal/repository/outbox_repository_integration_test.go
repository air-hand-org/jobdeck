package repository

import (
	"context"
	"net/url"
	"testing"
	"time"

	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/eventing"
	"jobdeck/services/job-api/internal/migrator"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestOutboxRepositoryReserveForPublishExecutionRequested_OnlyReservesDueQueuedExecutions(t *testing.T) {
	ctx := context.Background()
	pool := newMigratedTestPool(t, ctx)
	store := NewStore(pool)

	dueID := createRetryExecution(t, ctx, store, time.Now().Add(-time.Minute))
	createRetryExecution(t, ctx, store, time.Now().Add(time.Minute))
	runningID := createRetryExecution(t, ctx, store, time.Now().Add(-time.Minute))
	claimExecution(t, ctx, store, runningID)

	var got []domain.ExecutionId
	err := store.WithinOutboxTx(ctx, func(ctx context.Context, outbox eventing.OutboxRepository) error {
		var err error
		got, err = outbox.ReserveForPublishExecutionRequested(ctx, 10)
		return err
	})

	require.NoError(t, err)
	require.Equal(t, []domain.ExecutionId{dueID}, got)
}

func newMigratedTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(
		ctx,
		"postgres:18.4-bookworm",
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
	require.NoError(t, migrator.Up(ctx, uri))

	pool, err := pgxpool.New(ctx, uri)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func createRetryExecution(t *testing.T, ctx context.Context, store *Store, nextRunAt time.Time) domain.ExecutionId {
	t.Helper()

	ownerID := uuid.New()
	job := createRetryableJob(t, ctx, store, ownerID)
	execution := createExecution(t, ctx, store, ownerID, job.Id)
	markExecutionPublished(t, ctx, store, execution.Id)
	claimed := claimExecution(t, ctx, store, execution.Id)

	var retry domain.Execution
	err := store.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues app.ExecutionQueueRepository) error {
		failed, err := queues.ReportResult(
			ctx,
			execution.Id,
			domain.EXECUTION_STATUS_FAILED,
			claimed.LeaseToken,
			domain.ExecutionResult{
				Result: &domain.HttpExecutionResult{
					Error: &domain.HttpError{
						Type:    domain.HTTP_ERROR_TYPE_NETWORK,
						Message: "integration test failure",
					},
				},
			},
		)
		if err != nil {
			return err
		}

		retry, err = queues.CreateRetryUnsuccessful(ctx, failed.Id, nextRunAt)
		return err
	})
	require.NoError(t, err)
	return retry.Id
}

func createRetryableJob(t *testing.T, ctx context.Context, store *Store, ownerID domain.ActorId) domain.Job {
	t.Helper()

	target, err := url.ParseRequestURI("https://example.com")
	require.NoError(t, err)

	var job domain.Job
	err = store.WithinJobTx(ctx, func(ctx context.Context, jobs app.JobRepository) error {
		var err error
		job, err = jobs.Create(ctx, ownerID, "outbox integration test", domain.JobSpec{
			Type: domain.JOB_TYPE_HTTP_REQUEST,
			Config: &domain.HttpJobConfig{
				HttpMethod:          domain.HTTP_METHOD_GET,
				Url:                 *target,
				Timeout:             time.Second,
				ExpectedStatusCodes: domain.StatusCodes{Codes: []int32{200}},
				RetryOnNetworkError: true,
			},
			RetryPolicy: domain.RetryPolicy{
				MaxAttempts: 2,
				BackoffType: domain.BACKOFF_TYPE_FIXED,
			},
		})
		return err
	})
	require.NoError(t, err)
	return job
}

func createExecution(t *testing.T, ctx context.Context, store *Store, ownerID domain.ActorId, jobID domain.JobId) domain.Execution {
	t.Helper()

	var execution domain.Execution
	err := store.WithinExecutionTx(ctx, func(ctx context.Context, executions app.ExecutionRepository) error {
		var err error
		execution, err = executions.CreateForCurrentJob(ctx, ownerID, jobID, domain.EXECUTION_TRIGGER_MANUAL)
		return err
	})
	require.NoError(t, err)
	return execution
}

func markExecutionPublished(t *testing.T, ctx context.Context, store *Store, executionID domain.ExecutionId) {
	t.Helper()

	err := store.WithinOutboxTx(ctx, func(ctx context.Context, outbox eventing.OutboxRepository) error {
		return outbox.MarkAsPublishedExecutionRequested(ctx, []domain.ExecutionId{executionID})
	})
	require.NoError(t, err)
}

func claimExecution(t *testing.T, ctx context.Context, store *Store, executionID domain.ExecutionId) domain.ClaimedExecution {
	t.Helper()

	var claimed domain.ClaimedExecution
	err := store.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues app.ExecutionQueueRepository) error {
		var err error
		claimed, err = queues.Claim(ctx, executionID)
		return err
	})
	require.NoError(t, err)
	return claimed
}
