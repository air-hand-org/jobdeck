package transport

import (
	"testing"
	"time"

	jobv1 "jobdeck/gen/go/jobdeck/job/v1"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestJobSpecValidationRequiresPositiveHttpTimeout(t *testing.T) {
	validator, err := protovalidate.New()
	require.NoError(t, err)

	newSpec := func(timeout *durationpb.Duration) *jobv1.JobSpec {
		return &jobv1.JobSpec{
			Type: jobv1.JobType_JOB_TYPE_HTTP_REQUEST,
			Config: &jobv1.JobSpec_Http{
				Http: &jobv1.HttpJobConfig{
					Method:  jobv1.HttpMethod_HTTP_METHOD_GET,
					Url:     "https://example.com",
					Timeout: timeout,
				},
			},
			RetryPolicy: &jobv1.RetryPolicy{
				MaxAttempts: 1,
				BackoffType: jobv1.BackoffType_BACKOFF_TYPE_FIXED,
			},
		}
	}

	require.Error(t, validator.Validate(newSpec(nil)))
	require.Error(t, validator.Validate(newSpec(durationpb.New(0))))
	require.NoError(t, validator.Validate(newSpec(durationpb.New(time.Second))))
}
