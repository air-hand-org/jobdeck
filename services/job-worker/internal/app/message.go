package app

import jobv1 "jobdeck/gen/go/jobdeck/job/v1"

type WorkItem struct {
	Execution *jobv1.ClaimedExecution
}

type WorkResult struct {
	ExecutionID string
	LeaseToken  string
	Status      jobv1.ExecutionStatus
	Result      *jobv1.ExecutionResult
}
