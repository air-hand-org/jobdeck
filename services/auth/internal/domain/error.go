package domain

import "errors"

// ErrAuthClientNotFound indicates that an authentication client was not found.
var ErrAuthClientNotFound = errors.New("auth client is not found")
