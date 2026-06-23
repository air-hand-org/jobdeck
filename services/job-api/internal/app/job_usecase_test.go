package app

import (
	"context"
	"testing"

	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
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
