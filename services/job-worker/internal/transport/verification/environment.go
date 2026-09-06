// Package verification authenticates Pub/Sub push requests.
package verification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const metadataProjectIDURL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"

// NewFromEnvironment creates a verifier based on the deployment environment.
func NewFromEnvironment() (Verifier, error) {
	if token := os.Getenv("LOCAL_PUBSUB_VERIFICATION_TOKEN"); token != "" {
		return NewSharedTokenVerifier(token), nil
	}

	for _, name := range []string{"K_SERVICE", "K_REVISION", "K_CONFIGURATION"} {
		if os.Getenv(name) == "" {
			return nil, fmt.Errorf("%s is required when LOCAL_PUBSUB_VERIFICATION_TOKEN is not set", name)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := verifyGoogleMetadataServer(ctx, http.DefaultClient); err != nil {
		return nil, fmt.Errorf("cloud run environment verification failed: %w", err)
	}

	return NewPlatformVerifier(), nil
}

func verifyGoogleMetadataServer(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataProjectIDURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("metadata server returned %s", res.Status)
	}
	if res.Header.Get("Metadata-Flavor") != "Google" {
		return errors.New("metadata server response is missing Metadata-Flavor: Google")
	}

	projectID, err := io.ReadAll(io.LimitReader(res.Body, 4096))
	if err != nil {
		return err
	}
	if len(projectID) == 0 {
		return errors.New("metadata server returned an empty project id")
	}
	return nil
}
