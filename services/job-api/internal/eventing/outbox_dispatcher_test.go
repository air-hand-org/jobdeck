package eventing

import (
	"context"
	"testing"

	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxDispatcher_Dispatch(t *testing.T) {
	executionIDs := []domain.ExecutionId{uuid.New(), uuid.New()}
	publishErr := errors.New("publish failed")

	tests := []struct {
		name            string
		publishErrors   map[domain.ExecutionId]error
		wantMarked      []domain.ExecutionId
		wantDispatchErr error
	}{
		{
			name:       "marks all published executions",
			wantMarked: executionIDs,
		},
		{
			name: "marks only successfully published executions",
			publishErrors: map[domain.ExecutionId]error{
				executionIDs[1]: publishErr,
			},
			wantMarked: []domain.ExecutionId{executionIDs[0]},
		},
		{
			name: "rolls back when all publishes fail",
			publishErrors: map[domain.ExecutionId]error{
				executionIDs[0]: publishErr,
				executionIDs[1]: publishErr,
			},
			wantDispatchErr: publishErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outbox := &OutboxRepositoryMock{
				ReserveForPublishExecutionRequestedFunc: func(_ context.Context, maxSize int) ([]domain.ExecutionId, error) {
					assert.Equal(t, 10, maxSize)
					return executionIDs, nil
				},
				MarkAsPublishedExecutionRequestedFunc: func(_ context.Context, actualIDs []domain.ExecutionId) error {
					assert.Equal(t, tt.wantMarked, actualIDs)
					return nil
				},
			}
			store := &OutboxStoreMock{
				WithinOutboxTxFunc: func(ctx context.Context, fn func(context.Context, OutboxRepository) error) error {
					return fn(ctx, outbox)
				},
			}
			publisher := &ExecutionPublisherMock{
				PublishFunc: func(_ context.Context, executionID domain.ExecutionId) error {
					return tt.publishErrors[executionID]
				},
			}

			dispatcher := NewOutboxDispatcher(publisher, store)
			err := dispatcher.dispatch(context.Background())

			if tt.wantDispatchErr != nil {
				require.Error(t, err)
				assert.Empty(t, outbox.MarkAsPublishedExecutionRequestedCalls())
				return
			}

			require.NoError(t, err)
			assert.Len(t, outbox.MarkAsPublishedExecutionRequestedCalls(), 1)
		})
	}
}
