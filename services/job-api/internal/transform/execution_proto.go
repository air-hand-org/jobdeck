package transform

import (
	"jobdeck/services/job-api/internal/domain"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecutionIdFromProto parses an execution ID.
//
//nolint:revive // Existing internal API name is retained for compatibility.
func ExecutionIdFromProto(s string) (domain.ExecutionId, error) {
	return uuid.Parse(s)
}

// LeaseTokenFromProto parses a lease token.
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

func executionResultToProto(jobType domain.JobType, result domain.ExecutionResult) (*jobv1.ExecutionResult, error) {
	switch jobType {
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
		return nil, errors.Newf("unsupported job type: %v", jobType)
	}
}

// ExecutionToProto converts an execution to protobuf.
func ExecutionToProto(e domain.Execution) (*jobv1.Execution, error) {
	events := lo.Map(e.Events, func(e domain.ExecutionEvent, _ int) *jobv1.ExecutionEvent {
		return executionEventToProto(e)
	})
	var result *jobv1.ExecutionResult
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
