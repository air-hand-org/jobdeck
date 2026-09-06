// Package transport provides the job API's gRPC transport implementations.
package transport

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/auth"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/transform"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"

	"github.com/samber/lo"
)

// ExecutionServiceServer implements the execution gRPC service.
type ExecutionServiceServer struct {
	u app.ExecutionUsecase

	jobv1.UnsafeExecutionServiceServer
}

var _ jobv1.ExecutionServiceServer = (*ExecutionServiceServer)(nil)

// GetExecution returns an execution for the authenticated actor.
func (s *ExecutionServiceServer) GetExecution(ctx context.Context, req *jobv1.GetExecutionRequest) (*jobv1.GetExecutionResponse, error) {
	execID, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	exec, err := s.u.GetExecution(ctx, actor, execID)
	if err != nil {
		return nil, err
	}
	execProto, err := transform.ExecutionToProto(exec)
	if err != nil {
		return nil, err
	}
	return &jobv1.GetExecutionResponse{
		Execution: execProto,
	}, nil
}

// ListExecutions returns executions for a job for the authenticated actor.
func (s *ExecutionServiceServer) ListExecutions(ctx context.Context, req *jobv1.ListExecutionsRequest) (*jobv1.ListExecutionsResponse, error) {
	jobID, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	result, err := s.u.ListExecutions(ctx, actor, jobID, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	execsProto, err := lo.MapErr(result.Executions, func(e domain.Execution, _ int) (*jobv1.Execution, error) {
		return transform.ExecutionToProto(e)
	})
	if err != nil {
		return nil, err
	}
	return &jobv1.ListExecutionsResponse{
		Executions:    execsProto,
		NextPageToken: result.NextPageToken,
	}, nil
}

// DoExecution requests a manual execution of a job for the authenticated actor.
func (s *ExecutionServiceServer) DoExecution(ctx context.Context, req *jobv1.DoExecutionRequest) (*jobv1.DoExecutionResponse, error) {
	jobID, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	exec, err := s.u.RequestExecution(ctx, actor, jobID, domain.EXECUTION_TRIGGER_MANUAL)
	if err != nil {
		return nil, err
	}
	execProto, err := transform.ExecutionToProto(exec)
	if err != nil {
		return nil, err
	}
	return &jobv1.DoExecutionResponse{
		Execution: execProto,
	}, nil
}

// NewExecutionServiceServer creates an execution gRPC service implementation.
func NewExecutionServiceServer(u *app.ExecutionUsecase) *ExecutionServiceServer {
	return &ExecutionServiceServer{u: *u}
}
