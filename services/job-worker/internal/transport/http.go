package transport

import (
	"fmt"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/transport/handlers"
	"net/http"
)

func newHttpHandler(push_usecase *app.PushUsecase) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", handlers.NewPushHandler(push_usecase))
	return mux
}

func NewServer(port string, push_usecase *app.PushUsecase) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: newHttpHandler(push_usecase),
	}
}
