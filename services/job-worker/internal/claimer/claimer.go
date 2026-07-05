package claimer

import (
	"context"
	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
)

type Claimer struct {
	client      jobv1.ExecutionQueueServiceClient
	workerCount int
	idleDelay   time.Duration
}

func NewClaimer(conn *grpc.ClientConn, workerCount int, idleDelay time.Duration) *Claimer {
	return &Claimer{
		client:      jobv1.NewExecutionQueueServiceClient(conn),
		workerCount: workerCount,
		idleDelay:   idleDelay,
	}
}

func (c *Claimer) Run(ctx context.Context, jobs chan<- app.WorkItem, results <-chan app.WorkResult) error {
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

		res, err := c.client.ClaimExecutions(reqCtx, &jobv1.ClaimExecutionsRequest{
			MaxCount: int32(available),
		})
		if err != nil {
			return errors.Wrapf(err, "failed to claim executions: %d", available)
		}

		for _, e := range res.GetExecutions() {
			select {
			case jobs <- app.WorkItem{Execution: e}:
				in_flight++
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil {
					return errors.Wrap(err, "ctx err")
				}
				return nil
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

			if err := c.report(ctx, res); err != nil {
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
	// TODO: no limit when debugging
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := c.client.ReportExecutionResult(reqCtx, &jobv1.ReportExecutionResultRequest{
		ExecutionId: res.ExecutionID,
		LeaseToken:  res.LeaseToken,
		Status:      res.Status,
		Result:      res.Result,
	})
	if err != nil {
		return errors.Wrap(err, "failed to report execution result")
	}
	return nil
}
