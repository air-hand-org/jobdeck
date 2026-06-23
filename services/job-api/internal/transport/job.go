package transport

import (
	"context"
	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/transform"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// TODO: impl auth
var OwnerId domain.UserId = uuid.Max

type jobServiceServer struct {
	u app.JobUsecase

	jobv1.UnsafeJobServiceServer
}

var _ jobv1.JobServiceServer = (*jobServiceServer)(nil)

func (s *jobServiceServer) CreateJob(ctx context.Context, req *jobv1.CreateJobRequest) (*jobv1.CreateJobResponse, error) {
	if req == nil {
		return nil, errors.New("req is required")
	}
	spec, err := transform.JobSpecFromProto(req.Spec)
	if err != nil {
		return nil, err
	}
	job, err := s.u.CreateJob(ctx, OwnerId, spec)
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
	if req == nil {
		return nil, errors.New("req is required")
	}
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	spec, err := transform.JobSpecFromProto(req.GetSpec())
	if err != nil {
		return nil, err
	}
	job, err := s.u.UpdateJobSpec(ctx, OwnerId, job_id, spec)
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
	if req == nil {
		return nil, errors.New("req is required")
	}
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}

	job, err := s.u.SetJobEnabled(ctx, OwnerId, job_id, req.GetEnabled())
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
	if req == nil {
		return nil, errors.New("req is required")
	}
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	err = s.u.DeleteJob(ctx, OwnerId, job_id)
	if err != nil {
		return nil, err
	}
	return &jobv1.DeleteJobResponse{}, nil
}

func (s *jobServiceServer) GetJob(ctx context.Context, req *jobv1.GetJobRequest) (*jobv1.GetJobResponse, error) {
	if req == nil {
		return nil, errors.New("req is required")
	}
	job_id, err := transform.JobIdFromProto(req.GetJobId())
	if err != nil {
		return nil, err
	}
	job, err := s.u.GetJob(ctx, OwnerId, job_id)
	if err != nil {
		return nil, err
	}
	protoJob, err := transform.JobToProto(job)
	return &jobv1.GetJobResponse{
		Job: protoJob,
	}, nil
}

func (s *jobServiceServer) ListJobs(ctx context.Context, req *jobv1.ListJobsRequest) (*jobv1.ListJobsResponse, error) {
	if req == nil {
		return nil, errors.New("req is required")
	}
	jobs, err := s.u.ListJobs(ctx, OwnerId)
	if err != nil {
		return nil, err
	}
	protoJobs, err := lo.MapErr(jobs, func(j domain.Job, _ int) (*jobv1.Job, error) {
		return transform.JobToProto(j)
	})
	if err != nil {
		return nil, err
	}
	return &jobv1.ListJobsResponse{
		Jobs:          protoJobs,
		NextPageToken: "FIXME",
	}, nil
}

func NewJobServiceServer(u *app.JobUsecase) jobv1.JobServiceServer {
	return &jobServiceServer{u: *u}
}
