package handlers

import (
	"encoding/json"
	"fmt"
	jobv1 "jobdeck/gen/go/jobdeck/job/v1"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/transport/verification"
	"log/slog"
	"net/http"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/protobuf/proto"
)

type pushRequest struct {
	Message      pubsub.Message `json:"message"`
	Subscription string         `json:"subscription"`
}

type PushHandler struct {
	logger   *slog.Logger
	verifier verification.Verifier
	usecase  *app.PushUsecase
}

var _ http.Handler = (*PushHandler)(nil)

func (h *PushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := h.verifier.Verify(r); err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

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

func NewPushHandler(usecase *app.PushUsecase, verifier verification.Verifier) *PushHandler {
	return &PushHandler{
		logger:   slog.Default(),
		verifier: verifier,
		usecase:  usecase,
	}
}
