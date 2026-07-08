package transport

import (
	"context"
	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/transform"
	"log"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type executionQueueServiceServer struct {
	u app.ExecutionQueueUsecase

	jobv1.UnsafeExecutionQueueServiceServer
}

var _ jobv1.ExecutionQueueServiceServer = (*executionQueueServiceServer)(nil)

func (s *executionQueueServiceServer) ClaimExecutions(ctx context.Context, req *jobv1.ClaimExecutionsRequest) (*jobv1.ClaimExecutionsResponse, error) {
	claims, err := s.u.ClaimExecutions(ctx, req.GetMaxCount())
	if err != nil {
		log.Printf("%+v", err)
		return nil, status.Error(codes.Internal, "failed to claim executions")
	}
	claimsProto, err := lo.MapErr(claims, func(claim domain.ClaimedExecution, _ int) (*jobv1.ClaimedExecution, error) {
		return transform.ClaimedExecutionToProto(claim)
	})
	if err != nil {
		return nil, err
	}
	return &jobv1.ClaimExecutionsResponse{
		Executions: claimsProto,
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
	cmd := app.ReportExecutionResultCommand{
		ExecutionId:     exec_id,
		ExecutionResult: exec_result,
		LeaseToken:      lease_token,
	}
	err = s.u.ReportExecutionResult(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &jobv1.ReportExecutionResultResponse{}, nil
}

func NewExecutionQueueServiceServer(u *app.ExecutionQueueUsecase) *executionQueueServiceServer {
	return &executionQueueServiceServer{u: *u}
}
