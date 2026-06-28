package transform

import (
	"jobdeck/services/job-api/internal/domain"

	"github.com/google/uuid"
)

func ExecutionIdFromProto(s string) (domain.ExecutionId, error) {
	return uuid.Parse(s)
}

func LeaseTokenFromProto(s string) (domain.LeaseToken, error) {
	return uuid.Parse(s)
}
