package transform

import (
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ExecutionIdFromProto(s string) (domain.ExecutionId, error) {
	return uuid.Parse(s)
}

func LeaseTokenFromProto(s string) (domain.LeaseToken, error) {
	return uuid.Parse(s)
}

func executionEventToProto(e domain.ExecutionEvent) *jobv1.ExecutionEvent {
	return &jobv1.ExecutionEvent{
		Status:    ExecutionStatusToProto(e.Status),
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
}

func httpExecutionResultToProto(r domain.HttpExecutionResult) *jobv1.HttpExecutionResult {
	var httpErr *jobv1.HttpError
	if r.Error != nil {
		httpErr = &jobv1.HttpError{
			Type:    httpErrorTypeToProto(r.Error.Type),
			Message: r.Error.Message,
		}
	}
	return &jobv1.HttpExecutionResult{
		StatusCode: r.StatusCode,
		Error:      httpErr,
	}
}

func executionResultToProto(job_type domain.JobType, result domain.ExecutionResult) (*jobv1.ExecutionResult, error) {
	switch job_type {
	case domain.JOB_TYPE_HTTP_REQUEST:
		httpResult := result.HttpResult()
		if httpResult == nil {
			return nil, errors.New("http execution result is required")
		}
		return &jobv1.ExecutionResult{
			Result: &jobv1.ExecutionResult_Http{
				Http: httpExecutionResultToProto(*httpResult),
			},
		}, nil
	default:
		return nil, errors.Newf("unsupported job type: %v", job_type)
	}
}

func ExecutionToProto(e domain.Execution) (*jobv1.Execution, error) {
	events := lo.Map(e.Events, func(e domain.ExecutionEvent, _ int) *jobv1.ExecutionEvent {
		return executionEventToProto(e)
	})
	var result *jobv1.ExecutionResult = nil
	var err error
	if e.Result != nil {
		result, err = executionResultToProto(e.JobSpec.Type, *e.Result)
		if err != nil {
			return nil, err
		}
	}
	return &jobv1.Execution{
		Id:      e.Id.String(),
		JobId:   e.JobId.String(),
		Status:  ExecutionStatusToProto(e.Status),
		Trigger: executionTriggerToProto(e.Trigger),
		Events:  events,

		Attempt:     e.Attempt,
		MaxAttempts: e.MaxAttempts,
		Result:      result,
		CreatedAt:   timestamppb.New(e.CreatedAt),
		UpdatedAt:   timestamppb.New(e.UpdatedAt),
	}, nil
}
