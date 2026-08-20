package handlers

import (
	"encoding/json"
	"fmt"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"log/slog"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/protobuf/proto"
)

type pushRequest struct {
	Message      pubsub.Message `json:"message"`
	Subscription string         `json:"subscription"`
}

type PushHandler struct {
	logger                *slog.Logger
	pushVerificationToken string
	usecase               *app.PushUsecase
}

var _ http.Handler = (*PushHandler)(nil)

func (h *PushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if token, ok := r.URL.Query()["token"]; !ok || len(token) != 1 || token[0] != h.pushVerificationToken {
		http.Error(w, "Bad token", http.StatusBadRequest)
		return
	}

	// TODO: Validate Request token
	// authentication push subscription
	// authHeader := r.Header.Get("Authorization")
	// h.logger.InfoContext(
	// 	r.Context(),
	// 	"auth header debug delete after",
	// 	"value", authHeader,
	// )

	var pr pushRequest
	if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	requested := &jobv1.ExecutionRequested{}
	if err := proto.Unmarshal(pr.Message.Data, requested); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode message data: %v", err), http.StatusBadRequest)
		return
	}

	h.logger.InfoContext(
		r.Context(),
		"requested id",
		"id", requested.GetExecutionId(),
	)

	if err := h.usecase.ClaimAndEnqueue(r.Context(), requested.GetExecutionId()); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.logger.InfoContext(
		r.Context(),
		"claim and enqueued",
		"id", requested.GetExecutionId(),
	)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, http.StatusText(http.StatusCreated))
}

func NewPushHandler(usecase *app.PushUsecase) *PushHandler {
	return &PushHandler{
		logger:                slog.Default(),
		pushVerificationToken: os.Getenv("PUBSUB_VERIFICATION_TOKEN"),
		usecase:               usecase,
	}
}
