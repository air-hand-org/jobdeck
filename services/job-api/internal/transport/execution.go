package transport

import (
	"context"
	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/transform"

	"github.com/samber/lo"
)

type executionServiceServer struct {
	u app.ExecutionUsecase

	jobv1.UnsafeExecutionServiceServer
}

var _ jobv1.ExecutionServiceServer = (*executionServiceServer)(nil)

func (s *executionServiceServer) GetExecution(ctx context.Context, req *jobv1.GetExecutionRequest) (*jobv1.GetExecutionResponse, error) {
	exec_id, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, err
	}
	exec, err := s.u.GetExecution(ctx, ownerId, exec_id)
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

func (s *executionServiceServer) ListExecutions(ctx context.Context, req *jobv1.ListExecutionsRequest) (*jobv1.ListExecutionsResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	result, err := s.u.ListExecutions(ctx, ownerId, job_id, req.GetPageSize(), req.GetPageToken())
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

func (s *executionServiceServer) DoExecution(ctx context.Context, req *jobv1.DoExecutionRequest) (*jobv1.DoExecutionResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	exec, err := s.u.RequestExecution(ctx, ownerId, job_id, domain.EXECUTION_TRIGGER_MANUAL)
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

func NewExecutionServiceServer(u *app.ExecutionUsecase) *executionServiceServer {
	return &executionServiceServer{u: *u}
}
