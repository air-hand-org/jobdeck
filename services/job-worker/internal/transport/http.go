// Package transport provides the job-worker HTTP server.
package transport

import (
	"fmt"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/transport/handlers"
	"jobdeck/services/job-worker/internal/transport/verification"
	"net/http"
)

func newHTTPHandler(pushUsecase *app.PushUsecase, verifier verification.Verifier) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", handlers.NewPushHandler(pushUsecase, verifier))
	return mux
}

// NewServer creates the job-worker HTTP server.
func NewServer(port string, pushUsecase *app.PushUsecase, verifier verification.Verifier) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: newHTTPHandler(pushUsecase, verifier),
	}
}
