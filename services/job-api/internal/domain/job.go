package domain

import (
	"encoding/json"
	"net/url"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type JobId = uuid.UUID

type Job struct {
	Id      JobId   `json:"id"`
	OwnerId ActorId `json:"owner_id"`
	Enabled bool    `json:"enabled"`
	Name    string  `json:"name"`

	Spec JobSpec `json:"spec"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type JobType string

const (
	JOB_TYPE_HTTP_REQUEST JobType = "HTTP_REQUEST"
)

func (t JobType) Valid() bool {
	switch t {
	case JOB_TYPE_HTTP_REQUEST:
		return true
	default:
		return false
	}
}

type JobConfig interface {
	JobConfig()
}

type HttpHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CodeRange struct {
	from int32
	to   int32
}

func (r CodeRange) From() int32 {
	return r.from
}

func (r CodeRange) To() int32 {
	return r.to
}

func (r CodeRange) In(code int32) bool {
	return r.from <= code && code <= r.to
}

func (r CodeRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		From int32 `json:"from"`
		To   int32 `json:"to"`
	}{
		From: r.from,
		To:   r.to,
	})
}

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

func NewCodeRange(from, to int32) (CodeRange, error) {
	if from > to {
		return CodeRange{}, errors.New("range start must not exceed end")
	}
	return CodeRange{
		from: from,
		to:   to,
	}, nil
}

type StatusCodes struct {
	Codes  []int32     `json:"codes"`
	Ranges []CodeRange `json:"ranges"`
}

func (c StatusCodes) Contains(code int32) bool {
	if slices.Contains(c.Codes, code) {
		return true
	}
	return slices.ContainsFunc(c.Ranges, func(r CodeRange) bool {
		return r.In(code)
	})
}

type HttpMethod string

const (
	HTTP_METHOD_GET    HttpMethod = "GET"
	HTTP_METHOD_POST   HttpMethod = "POST"
	HTTP_METHOD_HEAD   HttpMethod = "HEAD"
	HTTP_METHOD_PUT    HttpMethod = "PUT"
	HTTP_METHOD_PATCH  HttpMethod = "PATCH"
	HTTP_METHOD_DELETE HttpMethod = "DELETE"
)

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

func (c HttpJobConfig) MarshalJSON() ([]byte, error) {
	type Alias HttpJobConfig
	return json.Marshal(&struct {
		Url string `json:"url"`
		*Alias
	}{
		Url:   c.Url.String(),
		Alias: (*Alias)(&c),
	})
}

func (c *HttpJobConfig) UnmarshalJSON(data []byte) error {
	type Alias HttpJobConfig
	var aux struct {
		Url string `json:"url"`
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

func (c *HttpJobConfig) JobConfig() {}

var _ JobConfig = (*HttpJobConfig)(nil)

type JobSpec struct {
	Type JobType `json:"type"`

	Config      JobConfig   `json:"config"`
	RetryPolicy RetryPolicy `json:"retry_policy"`
}

func (s JobSpec) HttpJobConfig() *HttpJobConfig {
	if c, ok := s.Config.(*HttpJobConfig); ok {
		return c
	}
	return nil
}

type BackoffType string

const (
	BACKOFF_TYPE_FIXED BackoffType = "FIXED"
)

type RetryPolicy struct {
	MaxAttempts int32       `json:"max_attempts"`
	BackoffType BackoffType `json:"backoff_type"`

	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
}
