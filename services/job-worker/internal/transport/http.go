package transport

import (
	"fmt"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/transport/handlers"
	"jobdeck/services/job-worker/internal/transport/verification"
	"net/http"
)

func newHttpHandler(push_usecase *app.PushUsecase, verifier verification.Verifier) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", handlers.NewPushHandler(push_usecase, verifier))
	return mux
}

func NewServer(port string, push_usecase *app.PushUsecase, verifier verification.Verifier) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: newHttpHandler(push_usecase, verifier),
	}
}
