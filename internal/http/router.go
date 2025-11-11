package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(h *Handler) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	r.HandleFunc("/ask", h.Ask).Methods(http.MethodPost)

	return r
}
