package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"

	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"
)

func timeToPGType(t time.Time) (pgtype.Timestamptz, error) {
	var to pgtype.Timestamptz
	if err := to.Scan(t); err != nil {
		return to, err
	}
	return to, nil
}

func timeFromPGType(t pgtype.Timestamptz) (time.Time, error) {
	if !t.Valid {
		return time.Time{}, errors.New("timestamptz is invalid")
	}
	value, err := t.Value()
	if err != nil {
		return time.Time{}, err
	}
	if x, ok := value.(time.Time); ok {
		return x, nil
	}
	return time.Time{}, fmt.Errorf("failed to transform to time: %v", value)
}

func unmarshalJobConfig(t domain.JobType, data []byte) (domain.JobConfig, error) {
	switch t {
	case domain.JOB_TYPE_HTTP_REQUEST:
		var cfg domain.HttpJobConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	default:
		return nil, fmt.Errorf("unsupported job type: %v", t)
	}
}

func retryPolicyFromParams(maxAttempts int32, backoffType string, initialDelay, maxDelay time.Duration) domain.RetryPolicy {
	return domain.RetryPolicy{
		MaxAttempts: maxAttempts,
		BackoffType: domain.BackoffType(backoffType),

		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
	}
}

func jobSpecFromParams(jobType string, config []byte, retryPolicy domain.RetryPolicy) (domain.JobSpec, error) {
	t := domain.JobType(jobType)
	cfg, err := unmarshalJobConfig(t, config)
	if err != nil {
		return domain.JobSpec{}, err
	}
	return domain.JobSpec{
		Type: t,

		Config:      cfg,
		RetryPolicy: retryPolicy,
	}, nil
}

func jobSpecFromDB(j jobdb.Job) (domain.JobSpec, error) {
	retryPolicy := retryPolicyFromParams(
		j.RetryMaxAttempts,
		j.RetryBackoffType,
		time.Duration(j.RetryInitialDelayMs)*time.Millisecond,
		time.Duration(j.RetryMaxDelayMs)*time.Millisecond,
	)
	return jobSpecFromParams(j.Type, j.Config, retryPolicy)
}

func jobFromDB(j jobdb.Job) (domain.Job, error) {
	spec, err := jobSpecFromDB(j)
	if err != nil {
		return domain.Job{}, err
	}

	createdAt, err := timeFromPGType(j.CreatedAt)
	if err != nil {
		return domain.Job{}, err
	}

	updatedAt, err := timeFromPGType(j.UpdatedAt)
	if err != nil {
		return domain.Job{}, err
	}

	return domain.Job{
		Id:      j.JobID,
		OwnerId: j.OwnerID,
		Enabled: j.Enabled,
		Name:    j.Name,

		Spec:      spec,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func executionResultFromParams(jobType domain.JobType, result []byte) (domain.Result, error) {
	switch jobType {
	case domain.JOB_TYPE_HTTP_REQUEST:
		var httpResult domain.HttpExecutionResult
		if err := json.Unmarshal(result, &httpResult); err != nil {
			return nil, err
		}
		return &httpResult, nil
	default:
		return nil, fmt.Errorf("unsupported job type: %v", jobType)
	}
}

func executionFromDB(e jobdb.Execution) (domain.Execution, error) {
	retryPolicy := retryPolicyFromParams(
		e.JobRetryMaxAttempts,
		e.JobRetryBackoffType,
		time.Duration(e.JobRetryInitialDelayMs)*time.Millisecond,
		time.Duration(e.JobRetryMaxDelayMs)*time.Millisecond,
	)
	spec, err := jobSpecFromParams(e.JobType, e.JobConfig, retryPolicy)
	if err != nil {
		return domain.Execution{}, err
	}
	var result *domain.ExecutionResult
	if len(e.Result) > 0 {
		res, err := executionResultFromParams(domain.JobType(e.JobType), e.Result)
		if err != nil {
			return domain.Execution{}, err
		}
		result = &domain.ExecutionResult{
			Result: res,
		}
	}
	leaseToken := domain.LeaseToken{}
	if e.LeaseToken != nil {
		leaseToken = *e.LeaseToken
	}

	return domain.Execution{
		Id:      e.ExecutionID,
		JobId:   e.JobID,
		JobSpec: spec,

		Trigger:     domain.ExecutionTrigger(e.TriggerType),
		Status:      domain.ExecutionStatus(e.CurrentStatus),
		Events:      nil,
		Attempt:     e.Attempt,
		MaxAttempts: e.JobRetryMaxAttempts,

		LeaseToken:     leaseToken,
		LeaseExpiresAt: e.LeaseExpiresAt.Time,
		NextRunAt:      e.NextRunAt.Time,

		Result: result,

		CreatedAt: e.CreatedAt.Time,
		UpdatedAt: e.UpdatedAt.Time,
	}, nil
}

func claimExecutionFromDB(e jobdb.JobExecutionClaimDetail) (domain.ClaimedExecution, error) {
	// FIXME
	retryPolicy := domain.RetryPolicy{}
	spec, err := jobSpecFromParams(e.JobType, e.JobConfig, retryPolicy)
	if err != nil {
		return domain.ClaimedExecution{}, err
	}
	return domain.ClaimedExecution{
		Id:    e.ExecutionID,
		JobId: e.JobID,

		Attempt:        e.Attempt,
		LeaseToken:     *e.LeaseToken,
		LeaseExpiresAt: e.LeaseExpiresAt.Time,
		JobSpec:        spec,
	}, nil
}

func executionEventsFromDB(events []jobdb.JobExecutionEvent) ([]domain.ExecutionEvent, error) {
	return lo.MapErr(events, func(ee jobdb.JobExecutionEvent, _ int) (domain.ExecutionEvent, error) {
		createdAt, err := timeFromPGType(ee.CreatedAt)
		if err != nil {
			return domain.ExecutionEvent{}, err
		}
		return domain.ExecutionEvent{
			CreatedAt: createdAt,
			Status:    domain.ExecutionStatus(ee.Status),
		}, nil
	})
}

func actorFromDB(actor jobdb.JobActor) (domain.Actor, error) {
	return domain.NewActor(
		actor.ActorID,
		actor.Active,
		domain.ActorType(actor.Type),
		domain.RoleName(actor.Role.String),
	)
}
