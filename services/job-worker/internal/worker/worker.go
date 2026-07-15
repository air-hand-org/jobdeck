package worker

import (
	"context"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"

	"github.com/cockroachdb/errors"
)

type executor interface {
	Exec(context.Context, app.WorkItem) (app.WorkResult, error)
}

func executorFactory(item app.WorkItem) (executor, error) {
	job_type := item.Execution.GetJobSpec().GetType()
	switch job_type {
	case jobv1.JobType_JOB_TYPE_HTTP_REQUEST:
		return newHttpExecutor(item), nil
	default:
		return nil, errors.New("unsupported job type")
	}
}

func Run(ctx context.Context, id int, jobs <-chan app.WorkItem, results chan<- app.WorkResult) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}
			return nil
		case job, ok := <-jobs:
			if !ok {
				return nil
			}
			executor, err := executorFactory(job)
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}
			result, err := executor.Exec(ctx, job)
			if err != nil {
				return errors.Wrap(err, "error on worker.Run")
			}
			select {
			case results <- result:
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil {
					return errors.Wrap(err, "error on worker.Run")
				}
				return nil
			}
		}
	}
}
