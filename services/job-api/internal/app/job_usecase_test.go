package app

import (
	"context"
	"net/url"
	"testing"
	"time"

	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestJobUsecase_CreateJob_RepositoryError(t *testing.T) {
	expectedErr := errors.New("failed to create job")
	ownerID := domain.UserId(uuid.Max)
	spec := domain.JobSpec{}

	repo := &JobRepositoryMock{
		CreateFunc: func(_ context.Context, actualOwnerID domain.UserId, actualSpec domain.JobSpec) (domain.Job, error) {
			assert.Equal(t, ownerID, actualOwnerID)
			assert.Equal(t, spec, actualSpec)
			return domain.Job{}, expectedErr
		},
	}

	u := NewJobUsecase(repo)
	_, err := u.CreateJob(context.Background(), ownerID, spec)

	assert.ErrorIs(t, err, expectedErr)
	assert.Len(t, repo.CreateCalls(), 1)
}

func TestJobUsecase_ListJobs_Empty(t *testing.T) {
	owner_id := domain.UserId(uuid.Max)
	repo := &JobRepositoryMock{
		ListByOwnerFunc: func(_ context.Context, actualOwnerId domain.UserId) ([]domain.Job, error) {
			assert.Equal(t, owner_id, actualOwnerId)
			return []domain.Job{}, nil
		},
	}

	u := NewJobUsecase(repo)
	jobs, err := u.ListJobs(context.Background(), owner_id)

	assert.NoError(t, err)
	assert.Empty(t, jobs)
}

func TestJobUsecase_SetJobEnabled(t *testing.T) {
	owner_id := domain.UserId(uuid.Max)
	job_id := domain.JobId(uuid.Max)

	tests := []struct {
		testName string
		enabled bool
	}{
		{
			testName: "enable",
			enabled: true,
		},
		{
			testName: "disable",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			repo := &JobRepositoryMock{
				SetEnabledFunc: func(_ context.Context, actualOwnerId domain.UserId, actualJobId domain.JobId, actualEnabled bool) (domain.Job, error) {
					assert.Equal(t, owner_id, actualOwnerId)
					assert.Equal(t, job_id, actualJobId)
					assert.Equal(t, tt.enabled, actualEnabled)
					return domain.Job{}, nil
				},
			}
			u := NewJobUsecase(repo)

			u.SetJobEnabled(context.Background(), owner_id, job_id, tt.enabled)
			assert.Len(t, repo.SetEnabledCalls(), 1)
		})
	}
}

func TestJobUsecase_UpdateJobSpec_ChangeName(t *testing.T) {
	owner_id := domain.UserId(uuid.Max)
	job_id := domain.JobId(uuid.Max)
	spec := validJobSpec()
	updatedSpec := validJobSpec()
	assert.Equal(t, spec, updatedSpec)
	repo := &JobRepositoryMock{
		UpdateSpecFunc: func(_ context.Context, actualOwnerId domain.UserId, actualJobId domain.JobId, actualSpec domain.JobSpec) (domain.Job, error) {
			assert.Equal(t, owner_id, actualOwnerId)
			assert.Equal(t, job_id, actualJobId)
			assert.Equal(t, updatedSpec, actualSpec)
			return domain.Job{}, nil
		},
	}

	u := NewJobUsecase(repo)

	expectedJobName := "Updated Job Name"
	updatedSpec.Name = expectedJobName
	u.UpdateJobSpec(context.Background(), owner_id, job_id, updatedSpec)
	assert.Len(t, repo.UpdateSpecCalls(), 1)
}

func validJobSpec() domain.JobSpec {
	return domain.JobSpec{
		Name: "FooBar",
		Type: domain.JOB_TYPE_HTTP_REQUEST,

		Config: &domain.HttpJobConfig{
			HttpMethod: domain.HTTP_METHOD_GET,
			Url:        *lo.Must(url.Parse("https://www.example.com/")),
			Headers: []domain.HttpHeader{
				{
					Name:  "Content-Type",
					Value: "application/json",
				},
			},
			Body:    []byte("hello world"),
			Timeout: 10 * time.Second,
			ExpectedStatusCodes: domain.StatusCodes{
				Codes: []int32{
					200,
				},
			},
			MaxResponseBodyBytes: 100 * 1024,
			FollowRedirects:      false,

			RetryOnTimeout:      true,
			RetryOnNetworkError: true,
			RetryStatusCodes: domain.StatusCodes{
				Ranges: []domain.CodeRange{
					lo.Must(domain.NewCodeRange(400, 499)),
					lo.Must(domain.NewCodeRange(500, 599)),
				},
			},
		},

		RetryPolicy: domain.RetryPolicy{
			MaxAttempts: 1,
			BackoffType: domain.BACKOFF_TYPE_FIXED,

			InitialDelay: 1 * time.Second,
			MaxDelay:     5 * time.Second,
		},
	}
}
