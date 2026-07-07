package domain

import "errors"

var ErrJobNotFound = errors.New("job is not found")
var ErrExecutionNotFound = errors.New("execution is not found")

var ErrExecutionNotReportable = errors.New("execution is not reportable")
