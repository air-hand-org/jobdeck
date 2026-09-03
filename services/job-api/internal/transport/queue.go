package transport

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/auth"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/transform"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type executionQueueServiceServer struct {
	u app.ExecutionQueueUsecase

	jobv1.UnsafeExecutionQueueServiceServer
}

var _ jobv1.ExecutionQueueServiceServer = (*executionQueueServiceServer)(nil)

func (s *executionQueueServiceServer) ClaimExecution(ctx context.Context, req *jobv1.ClaimExecutionRequest) (*jobv1.ClaimExecutionResponse, error) {
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	exec_id, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, err
	}
	claim, err := s.u.ClaimExecution(ctx, actor, exec_id)
	if err != nil {
		return nil, err
	}
	claimProto, err := transform.ClaimedExecutionToProto(claim)
	if err != nil {
		return nil, err
	}
	return &jobv1.ClaimExecutionResponse{
		Execution: claimProto,
	}, nil
}

func (s *executionQueueServiceServer) ReportExecutionResult(ctx context.Context, req *jobv1.ReportExecutionResultRequest) (*jobv1.ReportExecutionResultResponse, error) {
	exec_id, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform execution id: %v", err)
	}
	exec_result, err := transform.ExecutionResultFromProto(req.GetResult())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform result: %v", err)
	}
	lease_token, err := transform.LeaseTokenFromProto(req.GetLeaseToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform token: %v", err)
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	cmd := app.ReportExecutionResultCommand{
		ExecutionId:     exec_id,
		ExecutionResult: exec_result,
		LeaseToken:      lease_token,
	}
	err = s.u.ReportExecutionResult(ctx, actor, cmd)
	if err != nil {
		return nil, err
	}
	return &jobv1.ReportExecutionResultResponse{}, nil
}

func NewExecutionQueueServiceServer(u *app.ExecutionQueueUsecase) *executionQueueServiceServer {
	return &executionQueueServiceServer{u: *u}
}
