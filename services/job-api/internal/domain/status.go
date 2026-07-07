package domain

import "github.com/cockroachdb/errors"

func determineExecutionStatus(spec JobSpec, result ExecutionResult) (ExecutionStatus, error) {
	switch spec.Type {
	case JOB_TYPE_HTTP_REQUEST:
		return determineHttpExecutionStatus(spec.HttpJobConfig(), result.HttpResult())
	default:
		return "", errors.Newf("unsupported job type %s", spec.Type)
	}
}

func determineHttpExecutionStatus(cfg *HttpJobConfig, result *HttpExecutionResult) (ExecutionStatus, error) {
	if cfg == nil || result == nil {
		return "", errors.New("cfg and result are required")
	}
	if result.Error != nil {
		return EXECUTION_STATUS_FAILED, nil
	}

	if !cfg.ExpectedStatusCodes.Contains(result.StatusCode) {
		return EXECUTION_STATUS_FAILED, nil
	}

	return EXECUTION_STATUS_SUCCEEDED, nil
}
