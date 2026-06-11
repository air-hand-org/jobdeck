package transform

//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.JobType -proto jobdeck/gen/jobdeck/job/v1.JobType -name jobType -output job_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.BackoffType -proto jobdeck/gen/jobdeck/job/v1.BackoffType -name retryPolicyBackoffType -output backoff_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.HttpMethod -proto jobdeck/gen/jobdeck/job/v1.HttpMethod -name httpMethod -output http_method_enum_gen.go
