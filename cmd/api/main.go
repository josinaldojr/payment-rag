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

	addr := ":" + cfg.Port
	log.Printf("API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
