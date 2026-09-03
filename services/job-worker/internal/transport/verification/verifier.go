package verification

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// Verifier authenticates an HTTP request.
type Verifier interface {
	Verify(*http.Request) error
}

// SharedTokenVerifier authenticates requests with a shared query token.
type SharedTokenVerifier struct {
	token string
}

// NewSharedTokenVerifier creates a shared-token verifier.
func NewSharedTokenVerifier(token string) *SharedTokenVerifier {
	return &SharedTokenVerifier{token: token}
}

// Verify checks the shared token in r.
func (v *SharedTokenVerifier) Verify(r *http.Request) error {
	token, ok := r.URL.Query()["token"]
	if !ok || len(token) != 1 || subtle.ConstantTimeCompare([]byte(token[0]), []byte(v.token)) != 1 {
		return errors.New("invalid push verification token")
	}
	return nil
}

// PlatformVerifier delegates authentication to the hosting platform.
type PlatformVerifier struct{}

// NewPlatformVerifier creates a platform verifier.
func NewPlatformVerifier() *PlatformVerifier {
	return &PlatformVerifier{}
}

// Verify intentionally does nothing. In production, Cloud Run IAM authenticates
// Pub/Sub push requests before they reach the application by validating the
// Pub/Sub OIDC token and the caller's roles/run.invoker permission.
//
// This verifier is only safe while the service is deployed behind that platform
// boundary. If job-worker is moved to another platform, provide an equivalent
// ingress authentication layer or replace this verifier with one that performs
// request authentication itself.
func (v *PlatformVerifier) Verify(*http.Request) error {
	return nil
}
