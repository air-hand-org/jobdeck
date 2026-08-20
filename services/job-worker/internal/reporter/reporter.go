package reporter

import (
	"context"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"log/slog"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type Reporter struct {
	client jobv1.ExecutionQueueServiceClient
	logger *slog.Logger
	tracer trace.Tracer
}

func NewReporter(conn *grpc.ClientConn, logger *slog.Logger) *Reporter {
	return &Reporter{
		client: jobv1.NewExecutionQueueServiceClient(conn),
		logger: logger,
		tracer: otel.Tracer("job-worker/reporter"),
	}
}

func (r *Reporter) Run(ctx context.Context, results <-chan app.QueuedResult) error {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return errors.Wrap(err, "ctx err")
			}
			return nil
		case res, ok := <-results:
			if !ok {
				return nil
			}
			reportCtx := trace.ContextWithSpanContext(ctx, res.ParentSpan)
			if err := r.report(reportCtx, res.Result); err != nil {
				// TODO: retry
				return errors.Wrap(err, "failed to report")
			}
		}
	}
}

func (r *Reporter) report(ctx context.Context, res app.WorkResult) error {
	ctx, span := r.tracer.Start(ctx, "queue.report")
	defer span.End()
	// TODO: no limit when debugging
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := r.client.ReportExecutionResult(reqCtx, &jobv1.ReportExecutionResultRequest{
		ExecutionId: res.ExecutionID,
		LeaseToken:  res.LeaseToken,
		Result:      res.Result,
	})
	if err != nil {
		err := errors.Wrap(err, "failed to report execution result")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	r.logger.InfoContext(
		ctx,
		"succeeded to report",
		"id", res.ExecutionID,
	)
	return nil
}
