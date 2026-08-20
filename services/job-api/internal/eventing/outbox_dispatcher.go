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

type OutboxDispatcher struct {
	ep          ExecutionPublisher
	s           OutboxStore
	logger      *slog.Logger
	triggerChan chan struct{}
}

var _ app.OutboxDispatcher = (*OutboxDispatcher)(nil)

func (d *OutboxDispatcher) Trigger(ctx context.Context) error {
	select {
	case d.triggerChan <- struct{}{}:
	default:
	}
	return nil
}

func (d *OutboxDispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.triggerChan:
			d.dispatch(ctx)
		case <-ticker.C:
			d.dispatch(ctx)
		}
	}
}

func (d *OutboxDispatcher) dispatch(ctx context.Context) error {
	err := d.s.WithinOutboxTx(ctx, func(ctx context.Context, outbox OutboxRepository) error {
		exec_id_list, err := outbox.ReserveForPublishExecutionRequested(ctx, 10)
		if err != nil {
			return err
		}
		if len(exec_id_list) == 0 {
			return nil
		}
		d.logger.InfoContext(
			ctx,
			"publishing execution ids",
			"exec_id_list", exec_id_list,
		)
		succeeded_list := lo.FilterMap(exec_id_list, func(exec_id domain.ExecutionId, _ int) (domain.ExecutionId, bool) {
			pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
			defer pubCancel()
			if err := d.ep.Publish(pubCtx, exec_id); err != nil {
				d.logger.WarnContext(
					ctx,
					"failed to publish",
					"id", exec_id,
					"error", err,
				)
				return exec_id, false
			}
			return exec_id, true
		})
		if len(succeeded_list) == 0 {
			return errors.New("failed to publish")
		}
		d.logger.InfoContext(
			ctx,
			"succeeded to publish",
			"exec_id_list", succeeded_list,
		)
		if err := outbox.MarkAsPublishedExecutionRequested(ctx, succeeded_list); err != nil {
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

func NewOutboxDispatcher(publisher ExecutionPublisher, store OutboxStore) *OutboxDispatcher {
	return &OutboxDispatcher{
		ep:          publisher,
		s:           store,
		logger:      slog.Default(),
		triggerChan: make(chan struct{}, 1),
	}
}
