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
	actor, _ := domain.NewActor(domain.ActorId(uuid.Max), true, domain.ActorTypeUser, domain.RoleUser)
	name := "Example Job"
	spec := domain.JobSpec{}

	repo := &JobRepositoryMock{
		CreateFunc: func(_ context.Context, actualOwnerID domain.ActorId, actualName string, actualSpec domain.JobSpec) (domain.Job, error) {
			assert.Equal(t, actor.ID, actualOwnerID)
			assert.Equal(t, name, actualName)
			assert.Equal(t, spec, actualSpec)
			return domain.Job{}, expectedErr
		},
	}

	store := &JobStoreMock{
		JobsFunc: func() JobRepository {
			return repo
		},
		WithinJobTxFunc: func(ctx context.Context, fn func(ctx context.Context, r JobRepository) error) error {
			return fn(ctx, repo)
		},
	}

	u := NewJobUsecase(store)
	_, err := u.CreateJob(context.Background(), actor, name, spec)

	assert.ErrorIs(t, err, expectedErr)
	assert.Len(t, repo.CreateCalls(), 1)
}

func TestJobUsecase_ListJobs_Empty(t *testing.T) {
	actor, _ := domain.NewActor(domain.ActorId(uuid.Max), true, domain.ActorTypeUser, domain.RoleUser)
	page_size := int32(10)
	page_token := "foo/bar"
	repo := &JobRepositoryMock{
		ListByOwnerFunc: func(_ context.Context, actualOwnerId domain.ActorId, actualPageSize int32, actualPageToken string) (ListJobsResult, error) {
			assert.Equal(t, actor.ID, actualOwnerId)
			assert.Equal(t, page_size, actualPageSize)
			assert.Equal(t, page_token, actualPageToken)
			return ListJobsResult{
				Jobs: []domain.Job{},
				NextPageToken: "",
			}, nil
		},
	}

	store := &JobStoreMock{
		JobsFunc: func() JobRepository {
			return repo
		},
		WithinJobTxFunc: func(ctx context.Context, fn func(ctx context.Context, r JobRepository) error) error {
			return fn(ctx, repo)
		},
	}

	u := NewJobUsecase(store)
	result, err := u.ListJobs(context.Background(), actor, page_size, page_token)

	assert.NoError(t, err)
	assert.Empty(t, result.Jobs)
	assert.Empty(t, result.NextPageToken)
}

func TestJobUsecase_SetJobEnabled(t *testing.T) {
	actor, _ := domain.NewActor(domain.ActorId(uuid.Max), true, domain.ActorTypeUser, domain.RoleUser)
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
				SetEnabledFunc: func(_ context.Context, actualOwnerId domain.ActorId, actualJobId domain.JobId, actualEnabled bool) (domain.Job, error) {
					assert.Equal(t, actor.ID, actualOwnerId)
					assert.Equal(t, job_id, actualJobId)
					assert.Equal(t, tt.enabled, actualEnabled)
					return domain.Job{}, nil
				},
			}
			store := &JobStoreMock{
				JobsFunc: func() JobRepository {
					return repo
				},
				WithinJobTxFunc: func(ctx context.Context, fn func(ctx context.Context, r JobRepository) error) error {
					return fn(ctx, repo)
				},
			}

			u := NewJobUsecase(store)

			u.SetJobEnabled(context.Background(), actor, job_id, tt.enabled)
			assert.Len(t, repo.SetEnabledCalls(), 1)
		})
	}
}

func TestJobUsecase_UpdateJobSpec_ChangeName(t *testing.T) {
	actor, _ := domain.NewActor(domain.ActorId(uuid.Max), true, domain.ActorTypeUser, domain.RoleUser)
	job_id := domain.JobId(uuid.Max)
	spec := validJobSpec()
	updatedSpec := validJobSpec()
	assert.Equal(t, spec, updatedSpec)

	expectedJobName := "Updated Job Name"
	repo := &JobRepositoryMock{
		UpdateSpecFunc: func(_ context.Context, actualOwnerId domain.ActorId, actualJobId domain.JobId, actualSpec domain.JobSpec) (domain.Job, error) {
			assert.Equal(t, actor.ID, actualOwnerId)
			assert.Equal(t, job_id, actualJobId)
			assert.Equal(t, updatedSpec, actualSpec)
			return domain.Job{}, nil
		},
		RenameFunc: func(_ context.Context, actualOwnerId domain.ActorId, actualJobId domain.JobId, actualName string) error {
			assert.Equal(t, actor.ID, actualOwnerId)
			assert.Equal(t, job_id, actualJobId)
			assert.Equal(t, expectedJobName, actualName)
			return nil
		},
	}

	store := &JobStoreMock{
		JobsFunc: func() JobRepository {
			return repo
		},
		WithinJobTxFunc: func(ctx context.Context, fn func(ctx context.Context, r JobRepository) error) error {
			return fn(ctx, repo)
		},
	}

	u := NewJobUsecase(store)

	u.UpdateJobMetadata(context.Background(), actor, job_id, expectedJobName, updatedSpec)
	assert.Len(t, repo.UpdateSpecCalls(), 1)
}

func validJobSpec() domain.JobSpec {
	return domain.JobSpec{
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
