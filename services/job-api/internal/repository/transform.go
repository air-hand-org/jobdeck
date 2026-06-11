package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5/pgtype"

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

func jobSpecFromDB(j jobdb.Job) (domain.JobSpec, error) {
	t := domain.JobType(j.Type)
	cfg, err := unmarshalJobConfig(t, j.Config)
	if err != nil {
		return domain.JobSpec{}, err
	}
	return domain.JobSpec{
		Name: j.Name,
		Type: t,

		Config: cfg,
		RetryPolicy: domain.RetryPolicy{
			MaxAttempts: j.RetryMaxAttempts,
			BackoffType: domain.BackoffType(j.RetryBackoffType),

			InitialDelay: time.Duration(j.RetryInitialDelayMs) * time.Millisecond,
			MaxDelay:     time.Duration(j.RetryMaxDelayMs) * time.Millisecond,
		},
	}, nil
}

func jobFromDB(j jobdb.Job) (domain.Job, error) {
	spec, err := jobSpecFromDB(j)
	if err != nil {
		return domain.Job{}, err
	}

	created_at, err := timeFromPGType(j.CreatedAt)
	if err != nil {
		return domain.Job{}, err
	}

	updated_at, err := timeFromPGType(j.UpdatedAt)
	if err != nil {
		return domain.Job{}, err
	}

	return domain.Job{
		Id:      j.JobID,
		OwnerId: j.OwnerID,
		Enabled: j.Enabled,

		Spec:      spec,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
	}, nil
}
