package transform

import (
	"fmt"
	"net/url"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/domain"
)

func retryPolicyToProto(p domain.RetryPolicy) *jobv1.RetryPolicy {
	return &jobv1.RetryPolicy{
		MaxAttempts:  p.MaxAttempts,
		BackoffType:  retryPolicyBackoffTypeToProto(p.BackoffType),
		InitialDelay: durationpb.New(p.InitialDelay),
		MaxDelay:     durationpb.New(p.MaxDelay),
	}
}

func retryPolicyFromProto(p *jobv1.RetryPolicy) (domain.RetryPolicy, error) {
	if p == nil {
		return domain.RetryPolicy{}, errors.New("retry policy is required")
	}

	backoff_type, err := retryPolicyBackoffTypeFromProto(p.GetBackoffType())
	if err != nil {
		return domain.RetryPolicy{}, err
	}

	return domain.RetryPolicy{
		MaxAttempts: p.GetMaxAttempts(),
		BackoffType: backoff_type,

		InitialDelay: p.GetInitialDelay().AsDuration(),
		MaxDelay:     p.GetMaxDelay().AsDuration(),
	}, nil
}

func httpHeadersToProto(hs []domain.HttpHeader) []*jobv1.HttpHeader {
	return lo.Map(hs, func(h domain.HttpHeader, _ int) *jobv1.HttpHeader {
		return &jobv1.HttpHeader{
			Name:  h.Name,
			Value: h.Value,
		}
	})
}

func httpHeadersFromProto(hs []*jobv1.HttpHeader) ([]domain.HttpHeader, error) {
	return lo.MapErr(hs, func(h *jobv1.HttpHeader, _ int) (domain.HttpHeader, error) {
		if h == nil {
			return domain.HttpHeader{}, errors.New("HttpHeader is required")
		}
		return domain.HttpHeader{
			Name:  h.GetName(),
			Value: h.GetValue(),
		}, nil
	})
}

func statusCodesToProto(sc domain.StatusCodes) *jobv1.StatusCodes {
	return &jobv1.StatusCodes{
		Codes: sc.Codes,
		Ranges: lo.Map(sc.Ranges, func(r domain.CodeRange, _ int) *jobv1.StatusCodeRange {
			return &jobv1.StatusCodeRange{
				From: r.From(),
				To:   r.To(),
			}
		}),
	}
}

func statusCodesFromProto(sc *jobv1.StatusCodes) (domain.StatusCodes, error) {
	if sc == nil {
		return domain.StatusCodes{}, errors.New("status codes are required")
	}
	ranges, err := lo.MapErr(sc.Ranges, func(r *jobv1.StatusCodeRange, _ int) (domain.CodeRange, error) {
		if r == nil {
			return domain.CodeRange{}, errors.New("status code range is required")
		}
		return domain.NewCodeRange(r.GetFrom(), r.GetTo())
	})
	if err != nil {
		return domain.StatusCodes{}, err
	}
	return domain.StatusCodes{
		Codes:  sc.GetCodes(),
		Ranges: ranges,
	}, nil
}

func httpJobConfigToProto(cfg domain.HttpJobConfig) *jobv1.HttpJobConfig {
	return &jobv1.HttpJobConfig{
		Method:  httpMethodToProto(cfg.HttpMethod),
		Url:     cfg.Url.String(),
		Headers: httpHeadersToProto(cfg.Headers),
		Body:    cfg.Body,

		Timeout:             durationpb.New(cfg.Timeout),
		ExpectedStatusCodes: statusCodesToProto(cfg.ExpectedStatusCodes),

		MaxResponseBodyBytes: cfg.MaxResponseBodyBytes,
		FollowRedirects:      cfg.FollowRedirects,

		RetryOnTimeout:      cfg.RetryOnTimeout,
		RetryOnNetworkError: cfg.RetryOnNetworkError,
		RetryOnStatusCodes:  statusCodesToProto(cfg.RetryStatusCodes),
	}
}

func httpJobConfigFromProto(p *jobv1.HttpJobConfig) (domain.HttpJobConfig, error) {
	if p == nil {
		return domain.HttpJobConfig{}, errors.New("job spec http is required")
	}

	method, err := httpMethodFromProto(p.GetMethod())
	if err != nil {
		return domain.HttpJobConfig{}, err
	}

	url, err := url.ParseRequestURI(p.GetUrl())
	if err != nil {
		return domain.HttpJobConfig{}, err
	}

	headers, err := httpHeadersFromProto(p.GetHeaders())
	if err != nil {
		return domain.HttpJobConfig{}, err
	}

	expected_sc, err := statusCodesFromProto(p.GetExpectedStatusCodes())
	if err != nil {
		return domain.HttpJobConfig{}, err
	}

	retry_sc, err := statusCodesFromProto(p.GetRetryOnStatusCodes())
	if err != nil {
		return domain.HttpJobConfig{}, err
	}

	cfg := domain.HttpJobConfig{
		HttpMethod: method,
		Url:        *url,
		Headers:    headers,
		Body:       p.GetBody(),

		Timeout:             p.GetTimeout().AsDuration(),
		ExpectedStatusCodes: expected_sc,

		MaxResponseBodyBytes: p.GetMaxResponseBodyBytes(),
		FollowRedirects:      p.GetFollowRedirects(),

		RetryOnTimeout:      p.GetRetryOnTimeout(),
		RetryOnNetworkError: p.GetRetryOnNetworkError(),
		RetryStatusCodes:    retry_sc,
	}

	return cfg, nil
}

func JobIdFromProto(job_id string) (domain.JobId, error) {
	return uuid.Parse(job_id)
}

func JobSpecToProto(s domain.JobSpec) (*jobv1.JobSpec, error) {
	p := &jobv1.JobSpec{
		Type: jobTypeToProto(s.Type),

		RetryPolicy: retryPolicyToProto(s.RetryPolicy),
	}

	switch p.Type {
	case jobv1.JobType_JOB_TYPE_HTTP_REQUEST:
		p.Config = &jobv1.JobSpec_Http{
			Http: httpJobConfigToProto(*s.HttpJobConfig()),
		}
	default:
		return nil, fmt.Errorf("unsupported job type: %v", p.Type.String())
	}

	return p, nil
}

func JobSpecFromProto(p *jobv1.JobSpec) (domain.JobSpec, error) {
	if p == nil {
		return domain.JobSpec{}, errors.New("job spec is required")
	}

	job_type, err := jobTypeFromProto(p.GetType())
	if err != nil {
		return domain.JobSpec{}, err
	}

	retry_policy, err := retryPolicyFromProto(p.GetRetryPolicy())
	if err != nil {
		return domain.JobSpec{}, err
	}

	spec := domain.JobSpec{
		Type: job_type,

		RetryPolicy: retry_policy,
	}

	switch spec.Type {
	case domain.JOB_TYPE_HTTP_REQUEST:
		cfg, err := httpJobConfigFromProto(p.GetHttp())
		if err != nil {
			return domain.JobSpec{}, err
		}
		spec.Config = &cfg
	default:
		return domain.JobSpec{}, fmt.Errorf("unsupported job config: %v", spec.Type)
	}

	return spec, nil
}

func JobToProto(j domain.Job) (*jobv1.Job, error) {
	spec, err := JobSpecToProto(j.Spec)
	if err != nil {
		return nil, err
	}

	return &jobv1.Job{
		Id:      j.Id.String(),
		OwnerId: j.OwnerId.String(),
		Enabled: j.Enabled,
		Name:    j.Name,

		Spec: spec,

		CreatedAt: timestamppb.New(j.CreatedAt),
		UpdatedAt: timestamppb.New(j.UpdatedAt),
	}, nil
}

func JobFromProto(j *jobv1.Job) (domain.Job, error) {
	if j == nil {
		return domain.Job{}, errors.New("job is required")
	}

	id, err := uuid.Parse(j.GetId())
	if err != nil {
		return domain.Job{}, err
	}

	owner_id, err := uuid.Parse(j.GetOwnerId())
	if err != nil {
		return domain.Job{}, err
	}

	spec, err := JobSpecFromProto(j.GetSpec())
	if err != nil {
		return domain.Job{}, err
	}

	return domain.Job{
		Id:        id,
		OwnerId:   owner_id,
		Enabled:   j.GetEnabled(),
		Name:      j.GetName(),
		Spec:      spec,
		CreatedAt: j.GetCreatedAt().AsTime(),
		UpdatedAt: j.GetUpdatedAt().AsTime(),
	}, nil
}
