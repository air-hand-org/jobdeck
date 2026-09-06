// Package domain contains the job API domain model.
package domain

import (
	"encoding/json"
	"net/url"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

// JobId is the UUID of a job.
//
//nolint:revive // Retained for compatibility with the existing internal API.
type JobId = uuid.UUID

// Job represents a configured job.
type Job struct {
	//nolint:revive // JSON field names are part of the persisted API representation.
	Id JobId `json:"id"`
	//nolint:revive // JSON field names are part of the persisted API representation.
	OwnerId ActorId `json:"owner_id"`
	Enabled bool    `json:"enabled"`
	Name    string  `json:"name"`

	Spec JobSpec `json:"spec"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobType identifies the kind of job.
type JobType string

const (
	// JOB_TYPE_HTTP_REQUEST identifies an HTTP request job.
	//nolint:revive // Name mirrors the externally represented job type.
	JOB_TYPE_HTTP_REQUEST JobType = "HTTP_REQUEST"
)

// Valid reports whether the job type is supported.
func (t JobType) Valid() bool {
	switch t {
	case JOB_TYPE_HTTP_REQUEST:
		return true
	default:
		return false
	}
}

// JobConfig is the configuration for a job type.
type JobConfig interface {
	JobConfig()
}

// HttpHeader is an HTTP request header.
//
//nolint:revive // Existing domain name is retained for compatibility.
type HttpHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CodeRange is an inclusive range of status codes.
type CodeRange struct {
	from int32
	to   int32
}

// From returns the lower bound of the range.
func (r CodeRange) From() int32 {
	return r.from
}

// To returns the upper bound of the range.
func (r CodeRange) To() int32 {
	return r.to
}

// In reports whether code is within the range.
func (r CodeRange) In(code int32) bool {
	return r.from <= code && code <= r.to
}

// MarshalJSON encodes the range bounds.
func (r CodeRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		From int32 `json:"from"`
		To   int32 `json:"to"`
	}{
		From: r.from,
		To:   r.to,
	})
}

// UnmarshalJSON decodes the range bounds.
func (r *CodeRange) UnmarshalJSON(data []byte) error {
	var value struct {
		From int32 `json:"from"`
		To   int32 `json:"to"`
	}

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	parsed, err := NewCodeRange(value.From, value.To)
	if err != nil {
		return err
	}

	*r = parsed
	return nil
}

// NewCodeRange creates a range from from through to.
func NewCodeRange(from, to int32) (CodeRange, error) {
	if from > to {
		return CodeRange{}, errors.New("range start must not exceed end")
	}
	return CodeRange{
		from: from,
		to:   to,
	}, nil
}

// StatusCodes contains explicit codes and ranges.
type StatusCodes struct {
	Codes  []int32     `json:"codes"`
	Ranges []CodeRange `json:"ranges"`
}

// Contains reports whether code is listed or covered by a range.
func (c StatusCodes) Contains(code int32) bool {
	if slices.Contains(c.Codes, code) {
		return true
	}
	return slices.ContainsFunc(c.Ranges, func(r CodeRange) bool {
		return r.In(code)
	})
}

// HttpMethod identifies an HTTP request method.
//
//nolint:revive // Existing domain name is retained for compatibility.
type HttpMethod string

const (
	// HTTP_METHOD_GET and related constants are HTTP methods supported by HTTP jobs.
	//nolint:revive // Names mirror the externally represented HTTP methods.
	HTTP_METHOD_GET HttpMethod = "GET"
	//nolint:revive // Name mirrors the protobuf enum constant.
	HTTP_METHOD_POST HttpMethod = "POST"
	//nolint:revive // Name mirrors the externally represented HTTP method.
	HTTP_METHOD_HEAD HttpMethod = "HEAD"
	//nolint:revive // Name mirrors the externally represented HTTP method.
	HTTP_METHOD_PUT HttpMethod = "PUT"
	//nolint:revive // Name mirrors the externally represented HTTP method.
	HTTP_METHOD_PATCH HttpMethod = "PATCH"
	//nolint:revive // Name mirrors the externally represented HTTP method.
	HTTP_METHOD_DELETE HttpMethod = "DELETE"
)

// HttpJobConfig configures an HTTP request job.
//
//nolint:revive // Existing domain name is retained for compatibility.
type HttpJobConfig struct {
	HttpMethod HttpMethod   `json:"http_method"`
	Url        url.URL      `json:"url"`
	Headers    []HttpHeader `json:"headers"`
	Body       []byte       `json:"body"`

	Timeout             time.Duration `json:"timeout"`
	ExpectedStatusCodes StatusCodes   `json:"expected_status_codes"`

	MaxResponseBodyBytes int64 `json:"max_response_body_bytes"`
	FollowRedirects      bool  `json:"follow_redirects"`

	RetryOnTimeout      bool        `json:"retry_on_timeout"`
	RetryOnNetworkError bool        `json:"retry_on_network_error"`
	RetryStatusCodes    StatusCodes `json:"retry_status_codes"`
}

// MarshalJSON encodes the HTTP job configuration.
func (c HttpJobConfig) MarshalJSON() ([]byte, error) {
	type Alias HttpJobConfig
	return json.Marshal(&struct {
		Url string `json:"url"` //nolint:revive // JSON field name is persisted API compatibility.
		*Alias
	}{
		Url:   c.Url.String(),
		Alias: (*Alias)(&c),
	})
}

// UnmarshalJSON decodes the HTTP job configuration.
func (c *HttpJobConfig) UnmarshalJSON(data []byte) error {
	type Alias HttpJobConfig
	var aux struct {
		Url string `json:"url"` //nolint:revive // JSON field name is persisted API compatibility.
		Alias
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	parsed, err := url.ParseRequestURI(aux.Url)
	if err != nil {
		return err
	}
	decoded := HttpJobConfig(aux.Alias)
	decoded.Url = *parsed
	*c = decoded
	return nil
}

// JobConfig marks HttpJobConfig as a job configuration.
func (c *HttpJobConfig) JobConfig() {}

var _ JobConfig = (*HttpJobConfig)(nil)

// JobSpec describes a job and its retry policy.
type JobSpec struct {
	Type JobType `json:"type"`

	Config      JobConfig   `json:"config"`
	RetryPolicy RetryPolicy `json:"retry_policy"`
}

// HttpJobConfig returns the HTTP configuration, if present.
//
//nolint:revive // Existing method name is retained for compatibility.
func (s JobSpec) HttpJobConfig() *HttpJobConfig {
	if c, ok := s.Config.(*HttpJobConfig); ok {
		return c
	}
	return nil
}

// BackoffType identifies the retry backoff algorithm.
type BackoffType string

const (
	// BACKOFF_TYPE_FIXED identifies fixed retry delays.
	//nolint:revive // Name mirrors the externally represented retry strategy.
	BACKOFF_TYPE_FIXED BackoffType = "FIXED"
)

// RetryPolicy configures execution retries.
type RetryPolicy struct {
	MaxAttempts int32       `json:"max_attempts"`
	BackoffType BackoffType `json:"backoff_type"`

	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
}
