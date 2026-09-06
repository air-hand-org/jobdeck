// Package domain contains the job API domain model.
package domain

import "errors"

// Domain errors describe expected application failures.
var (
	ErrJobNotFound                 = errors.New("job is not found")
	ErrExecutionNotFound           = errors.New("execution is not found")
	ErrExecutionNotReportable      = errors.New("execution is not reportable")
	ErrExecutionHasAlreadyConsumed = errors.New("execution has already consumed")
	ErrActorNotFound               = errors.New("actor is not found")
	ErrActorInactive               = errors.New("actor is not active")
	ErrActorInvalidRole            = errors.New("actor role is invalid")
	ErrActorNotHasRole             = errors.New("actor doesn't have role")
	ErrTokenInvalid                = errors.New("token is invalid")
	ErrPermissionDenied            = errors.New("permission denied")
	ErrUnauthenticated             = errors.New("authenticate required")
)
