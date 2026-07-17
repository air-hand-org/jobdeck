package claimer

import (
	"context"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type Claimer struct {
	client      jobv1.ExecutionQueueServiceClient
	workerCount int
	idleDelay   time.Duration
	tracer      trace.Tracer
}

func NewClaimer(conn *grpc.ClientConn, workerCount int, idleDelay time.Duration) *Claimer {
	return &Claimer{
		client:      jobv1.NewExecutionQueueServiceClient(conn),
		workerCount: workerCount,
		idleDelay:   idleDelay,
		tracer:      otel.Tracer("job-worker/claimer"),
	}
}

func (c *Claimer) Run(ctx context.Context, items chan<- app.QueuedItem, results <-chan app.QueuedResult) error {
	ticker := time.NewTicker(c.idleDelay)
	defer ticker.Stop()

	in_flight := 0

	claim := func() error {
		available := c.workerCount - in_flight
		if available <= 0 {
			return nil
		}

		// TODO: no limit when debugging
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		claimCtx, claimSpan := c.tracer.Start(reqCtx, "queue.claim")
		res, err := c.client.ClaimExecutions(claimCtx, &jobv1.ClaimExecutionsRequest{
			MaxCount: int32(available),
		})
		err = func() error {
			defer claimSpan.End()
			if err != nil {
				claimSpan.RecordError(err)
				claimSpan.SetStatus(codes.Error, err.Error())
				return err
			}
			return nil
		}()
		if err != nil {
			return errors.Wrapf(err, "failed to claim executions: %d", available)
		}

		for _, e := range res.GetExecutions() {
			err := func() error {
				_, enqueueSpan := c.tracer.Start(
					ctx,
					"queue.enqueue",
					trace.WithLinks(trace.Link{
						SpanContext: claimSpan.SpanContext(),
					}),
				)
				defer enqueueSpan.End()
				enqueueSpan.SetAttributes(
					attribute.String("jobdeck.execution_id", e.ExecutionId),
				)
				select {
				case items <- app.QueuedItem{Item: app.WorkItem{Execution: e}, ParentSpan: enqueueSpan.SpanContext()}:
					in_flight++
					return nil
				case <-ctx.Done():
					err := ctx.Err()
					if err != nil {
						enqueueSpan.RecordError(err)
						enqueueSpan.SetStatus(codes.Error, err.Error())
						return errors.Wrap(err, "ctx err")
					}
					return nil
				}
			}()
			if err != nil {
				return err
			}
		}

		return nil
	}

	if err := claim(); err != nil && ctx.Err() == nil {
		// TODO
		return err
	}

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				return errors.Wrap(err, "ctx err")
			}
			return nil
		case res, ok := <-results:
			if !ok {
				return nil
			}
			if in_flight > 0 {
				in_flight--
			}

			reportCtx := trace.ContextWithSpanContext(ctx, res.ParentSpan)
			if err := c.report(reportCtx, res.Result); err != nil {
				// TODO: retry
				return errors.Wrap(err, "failed to report")
			}

			if err := claim(); err != nil {
				return errors.Wrap(err, "failed to claim")
			}
		case <-ticker.C:
			if err := claim(); err != nil {
				// TODO
				return errors.Wrap(err, "failed to claim")
			}
		}
	}
}

func (c *Claimer) report(ctx context.Context, res app.WorkResult) error {
	ctx, span := c.tracer.Start(ctx, "queue.report")
	defer span.End()
	// TODO: no limit when debugging
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := c.client.ReportExecutionResult(reqCtx, &jobv1.ReportExecutionResultRequest{
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
	return nil
}
