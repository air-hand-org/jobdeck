package verification

import (
	"net/http/httptest"
	"testing"
)

func TestSharedTokenVerifier(t *testing.T) {
	verifier := NewSharedTokenVerifier("secret")

	t.Run("accepts matching token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/?token=secret", nil)
		if err := verifier.Verify(req); err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		if err := verifier.Verify(req); err == nil {
			t.Fatal("Verify() error = nil, want error")
		}
	})

	t.Run("rejects wrong token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/?token=wrong", nil)
		if err := verifier.Verify(req); err == nil {
			t.Fatal("Verify() error = nil, want error")
		}
	})
}

func TestNewFromEnvironmentUsesLocalToken(t *testing.T) {
	t.Setenv("LOCAL_PUBSUB_VERIFICATION_TOKEN", "secret")
	t.Setenv("K_SERVICE", "")
	t.Setenv("K_REVISION", "")
	t.Setenv("K_CONFIGURATION", "")

	verifier, err := NewFromEnvironment()
	if err != nil {
		t.Fatalf("NewFromEnvironment() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/?token=secret", nil)
	if err := verifier.Verify(req); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}
