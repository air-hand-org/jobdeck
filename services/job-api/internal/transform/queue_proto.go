package transform

import (
	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ClaimedExecutionToProto(e domain.ClaimedExecution) (*jobv1.ClaimedExecution, error) {
	spec, err := JobSpecToProto(e.JobSpec)
	if err != nil {
		return nil, err
	}
	return &jobv1.ClaimedExecution{
		ExecutionId:    e.Id.String(),
		LeaseToken:     e.LeaseToken.String(),
		LeaseExpiresAt: timestamppb.New(e.LeaseExpiresAt),
		Attempt:        e.Attempt,
		JobId:          e.JobId.String(),
		JobSpec:        spec,
	}, nil
}

func ExecutionResultFromProto(p *jobv1.ExecutionResult) (domain.ExecutionResult, error) {
	if p == nil {
		return domain.ExecutionResult{}, errors.New("execution result is required")
	}
	switch r := p.Result.(type) {
	case *jobv1.ExecutionResult_Http:
		return domain.ExecutionResult{
			Result: &domain.HttpExecutionResult{
				StatusCode: r.Http.GetStatusCode(),
			},
			ErrorMessage: p.GetErrorMessage(),
		}, nil
	default:
		return domain.ExecutionResult{}, errors.New("unsupported execution result")
	}
}
