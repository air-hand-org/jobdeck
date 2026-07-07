package transform

//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.JobType -proto jobdeck/gen/jobdeck/job/v1.JobType -name jobType -output job_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.BackoffType -proto jobdeck/gen/jobdeck/job/v1.BackoffType -name retryPolicyBackoffType -output backoff_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.HttpMethod -proto jobdeck/gen/jobdeck/job/v1.HttpMethod -name httpMethod -output http_method_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.ExecutionStatus -proto jobdeck/gen/jobdeck/job/v1.ExecutionStatus -name ExecutionStatus -output execution_status_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.ExecutionTrigger -proto jobdeck/gen/jobdeck/job/v1.ExecutionTrigger -name executionTrigger -output execution_trigger_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.HttpErrorType -proto jobdeck/gen/jobdeck/job/v1.HttpErrorType -name httpErrorType -output http_error_type_enum_gen.go
