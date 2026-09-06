package worker

import (
	"context"
	"jobdeck/services/job-worker/internal/app"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"
	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type executor interface {
	Exec(context.Context, app.WorkItem) (app.WorkResult, error)
}

func executorFactory(item app.WorkItem) (executor, error) {
	jobType := item.Execution.GetJobSpec().GetType()
	switch jobType {
	case jobv1.JobType_JOB_TYPE_HTTP_REQUEST:
		return newHTTPExecutor(item)
	default:
		return nil, errors.New("unsupported job type")
	}
}

// Worker executes claimed jobs.
type Worker struct {
	tracer trace.Tracer
}

// NewWorker creates a worker.
func NewWorker() *Worker {
	return &Worker{
		tracer: otel.Tracer("job-worker/worker"),
	}
}

// Run processes work items until ctx is canceled or works is closed.
func (w Worker) Run(ctx context.Context, _ int, works <-chan app.QueuedItem, results chan<- app.QueuedResult) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}
			return nil
		case work, ok := <-works:
			if !ok {
				return nil
			}
			itemCtx := trace.ContextWithSpanContext(ctx, work.ParentSpan)
			_, dequeueSpan := w.tracer.Start(itemCtx, "queue.dequeue")
			dequeueSpan.End()
			executor, err := executorFactory(work.Item)
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}
			_, execSpan := w.tracer.Start(itemCtx, "exec.execute")
			result, err := executor.Exec(itemCtx, work.Item)
			err = func() error {
				defer execSpan.End()
				if err != nil {
					execSpan.RecordError(err)
					execSpan.SetStatus(codes.Error, err.Error())
					return err
				}
				return nil
			}()
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}

			reportQueueCtx := trace.ContextWithSpanContext(ctx, execSpan.SpanContext())
			_, reportQueueSpan := w.tracer.Start(reportQueueCtx, "queue.report_enqueue")
			defer reportQueueSpan.End()
			select {
			case results <- app.QueuedResult{Result: result, ParentSpan: reportQueueSpan.SpanContext()}:
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil {
					reportQueueSpan.RecordError(err)
					reportQueueSpan.SetStatus(codes.Error, err.Error())
					return errors.Wrap(err, "error on worker.Run")
				}
				return nil
			}
		}
	}
}
