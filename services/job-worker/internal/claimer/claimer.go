package claimer

import (
	"context"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Claimer struct {
	client jobv1.ExecutionQueueServiceClient
	items  chan<- app.QueuedItem
	logger *slog.Logger
	tracer trace.Tracer
}

var _ app.Claimer = (*Claimer)(nil)

func NewClaimer(conn *grpc.ClientConn, items chan<- app.QueuedItem, logger *slog.Logger) *Claimer {
	return &Claimer{
		client: jobv1.NewExecutionQueueServiceClient(conn),
		items:  items,
		logger: logger,
		tracer: otel.Tracer("job-worker/claimer"),
	}
}

func (c *Claimer) ClaimAndEnqueue(ctx context.Context, execution_id string) error {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	res, err := c.client.ClaimExecution(reqCtx, &jobv1.ClaimExecutionRequest{
		ExecutionId: execution_id,
	})
	cancel()
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition:
			c.logger.WarnContext(
				ctx,
				"invalid execution id",
				"id", execution_id,
				"error", err,
			)
			return nil
		}
		return err
	}

	e := res.GetExecution()
	_, enqueueSpan := c.tracer.Start(
		ctx,
		"queue.enqueue",
	)
	defer enqueueSpan.End()
	enqueueSpan.SetAttributes(
		attribute.String("jobdeck.execution_id", e.ExecutionId),
	)

	select {
	case c.items <- app.QueuedItem{Item: app.WorkItem{Execution: e}, ParentSpan: enqueueSpan.SpanContext()}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
