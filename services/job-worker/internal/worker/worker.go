package worker

import (
	"context"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type executor interface {
	Exec(context.Context, app.WorkItem) (app.WorkResult, error)
}

func executorFactory(item app.WorkItem) (executor, error) {
	job_type := item.Execution.GetJobSpec().GetType()
	switch job_type {
	case jobv1.JobType_JOB_TYPE_HTTP_REQUEST:
		return newHttpExecutor(item)
	default:
		return nil, errors.New("unsupported job type")
	}
}

type Worker struct {
	tracer trace.Tracer
}

func NewWorker() *Worker {
	return &Worker{
		tracer: otel.Tracer("job-worker/worker"),
	}
}

func (w Worker) Run(ctx context.Context, id int, works <-chan app.QueuedItem, results chan<- app.QueuedResult) error {
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
