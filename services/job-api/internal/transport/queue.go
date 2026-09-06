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

// ExecutionQueueServiceServer implements the execution queue gRPC service.
type ExecutionQueueServiceServer struct {
	u app.ExecutionQueueUsecase

	jobv1.UnsafeExecutionQueueServiceServer
}

var _ jobv1.ExecutionQueueServiceServer = (*ExecutionQueueServiceServer)(nil)

// ClaimExecution claims an execution for the authenticated actor.
func (s *ExecutionQueueServiceServer) ClaimExecution(ctx context.Context, req *jobv1.ClaimExecutionRequest) (*jobv1.ClaimExecutionResponse, error) {
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	execID, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, err
	}
	claim, err := s.u.ClaimExecution(ctx, actor, execID)
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

// ReportExecutionResult reports an execution result for the authenticated actor.
func (s *ExecutionQueueServiceServer) ReportExecutionResult(ctx context.Context, req *jobv1.ReportExecutionResultRequest) (*jobv1.ReportExecutionResultResponse, error) {
	execID, err := transform.ExecutionIdFromProto(req.GetExecutionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform execution id: %v", err)
	}
	execResult, err := transform.ExecutionResultFromProto(req.GetResult())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform result: %v", err)
	}
	leaseToken, err := transform.LeaseTokenFromProto(req.GetLeaseToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to transform token: %v", err)
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	cmd := app.ReportExecutionResultCommand{
		ExecutionId:     execID,
		ExecutionResult: execResult,
		LeaseToken:      leaseToken,
	}
	err = s.u.ReportExecutionResult(ctx, actor, cmd)
	if err != nil {
		return nil, err
	}
	return &jobv1.ReportExecutionResultResponse{}, nil
}

// NewExecutionQueueServiceServer creates an execution queue gRPC service implementation.
func NewExecutionQueueServiceServer(u *app.ExecutionQueueUsecase) *ExecutionQueueServiceServer {
	return &ExecutionQueueServiceServer{u: *u}
}
