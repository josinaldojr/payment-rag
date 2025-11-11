package rag

import "context"

type EmbeddingsClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type LLMClient interface {
	GenerateAnswer(ctx context.Context, question string, chunks []DocChunk, provider Provider) (string, error)
}
