// Package transform converts between domain and protobuf values.
package transform

//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.JobType -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.JobType -name jobType -output job_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.BackoffType -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.BackoffType -name retryPolicyBackoffType -output backoff_type_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.HttpMethod -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.HttpMethod -name httpMethod -output http_method_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.ExecutionStatus -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.ExecutionStatus -name ExecutionStatus -output execution_status_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.ExecutionTrigger -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.ExecutionTrigger -name executionTrigger -output execution_trigger_enum_gen.go
//go:generate go run jobdeck/services/job-api/internal/cmd/enumgen -domain jobdeck/services/job-api/internal/domain.HttpErrorType -proto github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1.HttpErrorType -name httpErrorType -output http_error_type_enum_gen.go
