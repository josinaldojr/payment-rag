package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/josinaldojr/payment-gateway-rag/internal/rag"
)

type Handler struct {
	ragService *rag.Service
}

func NewHandler(ragService *rag.Service) *Handler {
	return &Handler{ragService: ragService}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) Ask(w http.ResponseWriter, r *http.Request) {
	var req rag.AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	if req.Lang == "" {
    req.Lang = "auto"
	}	

	resp, err := h.ragService.Ask(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
