package domain

import "errors"

var ErrJobNotFound = errors.New("job is not found")

var ErrExecutionNotFound = errors.New("execution is not found")
var ErrExecutionNotReportable = errors.New("execution is not reportable")

var ErrActorNotFound = errors.New("actor is not found")
var ErrActorInactive = errors.New("actor is not active")
var ErrActorInvalidRole = errors.New("actor role is invalid")
var ErrActorNotHasRole = errors.New("actor doesn't have role")
var ErrTokenInvalid = errors.New("token is invalid")
var ErrPermissionDenied = errors.New("permission denied")
var ErrUnauthenticated = errors.New("authenticate required")
