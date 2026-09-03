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

type jobServiceServer struct {
	u app.JobUsecase

	jobv1.UnsafeJobServiceServer
}

var _ jobv1.JobServiceServer = (*jobServiceServer)(nil)

func (s *jobServiceServer) CreateJob(ctx context.Context, req *jobv1.CreateJobRequest) (*jobv1.CreateJobResponse, error) {
	spec, err := transform.JobSpecFromProto(req.Spec)
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	job, err := s.u.CreateJob(ctx, actor, req.GetName(), spec)
	if err != nil {
		return nil, err
	}
	p, err := transform.JobToProto(job)
	if err != nil {
		return nil, err
	}
	return &jobv1.CreateJobResponse{
		Job: p,
	}, nil
}

func (s *jobServiceServer) UpdateJob(ctx context.Context, req *jobv1.UpdateJobRequest) (*jobv1.UpdateJobResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	spec, err := transform.JobSpecFromProto(req.GetSpec())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	job, err := s.u.UpdateJobMetadata(ctx, actor, job_id, req.GetName(), spec)
	if err != nil {
		return nil, err
	}
	protoJob, err := transform.JobToProto(job)
	if err != nil {
		return nil, err
	}
	return &jobv1.UpdateJobResponse{
		Job: protoJob,
	}, nil
}

func (s *jobServiceServer) SetJobEnabled(ctx context.Context, req *jobv1.SetJobEnabledRequest) (*jobv1.SetJobEnabledResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}

	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	job, err := s.u.SetJobEnabled(ctx, actor, job_id, req.GetEnabled())
	if err != nil {
		return nil, err
	}
	protoJob, err := transform.JobToProto(job)
	if err != nil {
		return nil, err
	}
	return &jobv1.SetJobEnabledResponse{
		Job: protoJob,
	}, nil
}

func (s *jobServiceServer) DeleteJob(ctx context.Context, req *jobv1.DeleteJobRequest) (*jobv1.DeleteJobResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	err = s.u.DeleteJob(ctx, actor, job_id)
	if err != nil {
		return nil, err
	}
	return &jobv1.DeleteJobResponse{}, nil
}

func (s *jobServiceServer) GetJob(ctx context.Context, req *jobv1.GetJobRequest) (*jobv1.GetJobResponse, error) {
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	job, err := s.u.GetJob(ctx, actor, job_id)
	if err != nil {
		return nil, err
	}
	protoJob, err := transform.JobToProto(job)
	return &jobv1.GetJobResponse{
		Job: protoJob,
	}, nil
}

func (s *jobServiceServer) ListJobs(ctx context.Context, req *jobv1.ListJobsRequest) (*jobv1.ListJobsResponse, error) {
	actor, ok := auth.GetActorContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	result, err := s.u.ListJobs(ctx, actor, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	protoJobs, err := lo.MapErr(result.Jobs, func(j domain.Job, _ int) (*jobv1.Job, error) {
		return transform.JobToProto(j)
	})
	if err != nil {
		return nil, err
	}
	return &jobv1.ListJobsResponse{
		Jobs:          protoJobs,
		NextPageToken: result.NextPageToken,
	}, nil
}

func NewJobServiceServer(u *app.JobUsecase) *jobServiceServer {
	return &jobServiceServer{u: *u}
}
