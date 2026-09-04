package eventing

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"log/slog"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

const dispatchInterval = 3 * time.Second

// OutboxDispatcher dispatches pending outbox events.
type OutboxDispatcher struct {
	ep          ExecutionPublisher
	s           OutboxStore
	logger      *slog.Logger
	triggerChan chan struct{}
}

var _ app.OutboxDispatcher = (*OutboxDispatcher)(nil)

// Trigger requests an outbox dispatch.
func (d *OutboxDispatcher) Trigger(_ context.Context) error {
	select {
	case d.triggerChan <- struct{}{}:
	default:
	}
	return nil
}

// Run dispatches outbox events until the context is canceled.
func (d *OutboxDispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(dispatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.triggerChan:
			if err := d.dispatch(ctx); err != nil {
				d.logger.ErrorContext(ctx, "failed to dispatch triggered outbox", "error", err)
			}
		case <-ticker.C:
			if err := d.dispatch(ctx); err != nil {
				d.logger.ErrorContext(ctx, "failed to dispatch scheduled outbox", "error", err)
			}
		}
	}
}

func (d *OutboxDispatcher) dispatch(ctx context.Context) error {
	err := d.s.WithinOutboxTx(ctx, func(ctx context.Context, outbox OutboxRepository) error {
		execIDList, err := outbox.ReserveForPublishExecutionRequested(ctx, 10)
		if err != nil {
			return err
		}
		if len(execIDList) == 0 {
			return nil
		}
		d.logger.InfoContext(
			ctx,
			"publishing execution ids",
			"exec_id_list", execIDList,
		)
		succeededList := lo.FilterMap(execIDList, func(execID domain.ExecutionId, _ int) (domain.ExecutionId, bool) {
			pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
			defer pubCancel()
			if err := d.ep.Publish(pubCtx, execID); err != nil {
				d.logger.WarnContext(
					ctx,
					"failed to publish",
					"id", execID,
					"error", err,
				)
				return execID, false
			}
			return execID, true
		})
		if len(succeededList) == 0 {
			return errors.New("failed to publish")
		}
		d.logger.InfoContext(
			ctx,
			"succeeded to publish",
			"exec_id_list", succeededList,
		)
		if err := outbox.MarkAsPublishedExecutionRequested(ctx, succeededList); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		d.logger.ErrorContext(
			ctx,
			"failed to dispatch",
			"error", err,
		)
		return err
	}
	return nil
}

// NewOutboxDispatcher constructs an outbox dispatcher.
func NewOutboxDispatcher(publisher ExecutionPublisher, store OutboxStore) *OutboxDispatcher {
	return &OutboxDispatcher{
		ep:          publisher,
		s:           store,
		logger:      slog.Default(),
		triggerChan: make(chan struct{}, 1),
	}
}
