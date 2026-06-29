package repository

import (
	"encoding/base64"
	"encoding/json"
	"jobdeck/services/job-api/internal/domain"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type jobPageToken struct {
	CreatedAt time.Time    `json:"created_at"`
	JobId     domain.JobId `json:"job_id"`
}

func encodeJobPageToken(t jobPageToken) (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeJobPageToken(s string) (jobPageToken, error) {
	d, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return jobPageToken{}, err
	}
	var t jobPageToken
	if err := json.Unmarshal(d, &t); err != nil {
		return jobPageToken{}, err
	}
	if t.CreatedAt.IsZero() || t.JobId == uuid.Nil {
		return jobPageToken{}, errors.New("invalid page token")
	}
	return t, nil
}

type executionPageToken struct {
	CreatedAt   time.Time          `json:"created_at"`
	ExecutionId domain.ExecutionId `json:"execution_id"`
}

func encodeExecutionPageToken(t executionPageToken) (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeExecutionPageToken(s string) (executionPageToken, error) {
	d, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return executionPageToken{}, err
	}
	var t executionPageToken
	if err := json.Unmarshal(d, &t); err != nil {
		return executionPageToken{}, err
	}
	if t.CreatedAt.IsZero() || t.ExecutionId == uuid.Nil {
		return executionPageToken{}, errors.New("invalid page token")
	}
	return t, nil
}
