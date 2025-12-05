package main

import (
	"context"
	"log"
	"net/http"

	"github.com/josinaldojr/payment-gateway-rag/internal/config"
	"github.com/josinaldojr/payment-gateway-rag/internal/db"
	apphttp "github.com/josinaldojr/payment-gateway-rag/internal/http"
	"github.com/josinaldojr/payment-gateway-rag/internal/llm"
	"github.com/josinaldojr/payment-gateway-rag/internal/rag"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	repo := rag.NewPgRepository(pool)

	geminiClient, err := llm.NewGeminiClient(ctx)
	if err != nil {
		log.Fatalf("failed to init Gemini client: %v", err)
	}

	ragService := rag.NewService(repo, geminiClient, geminiClient)

	h := apphttp.NewHandler(ragService)
	router := apphttp.NewRouter(h)

	handler := corsMiddleware(router)

	addr := ":" + cfg.Port
	log.Printf("API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
