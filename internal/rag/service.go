package rag

import (
	"context"
	"errors"
	"strings"
	wl "github.com/abadojack/whatlanggo"
)

type Service struct {
	repo       Repository
	embeddings EmbeddingsClient
	llm        LLMClient
}

func NewService(repo Repository, embeddings EmbeddingsClient, llm LLMClient) *Service {
	return &Service{
		repo:       repo,
		embeddings: embeddings,
		llm:        llm,
	}
}

func (s *Service) Ask(ctx context.Context, req AskRequest) (*AskResponse, error) {
	q := strings.TrimSpace(req.Question)
	if q == "" {
		return nil, errors.New("question is required")
	}

	// Resolve provider
	provider := resolveProvider(req.Provider, q)
	if provider == "" {
		return nil, errors.New("could not infer provider (ex: use 'rede' ou 'entrepay')")
	}

	// Embedding da pergunta
	vec, err := s.embeddings.Embed(ctx, q)
	if err != nil {
		return nil, err
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}

	if req.Lang == "" || req.Lang == "auto" {
      req.Lang = detectLang(q) // nova função logo abaixo
  }

	// Busca vetorial
	chunks, err := s.repo.SearchSimilarChunks(ctx, provider, vec, topK)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return &AskResponse{
			Answer:   "Não encontrei nada na documentação indexada para essa pergunta.",
			Provider: provider,
			Sources:  []SourceRef{},
		}, nil
	}

	// Gera resposta final com LLM usando os chunks
	answer, err := s.llm.GenerateAnswer(ctx, q, chunks, provider, req.Lang)
	if err != nil {
		return nil, err
	}

	// Monta fontes
	sources := make([]SourceRef, 0, len(chunks))
	for _, c := range chunks {
		sources = append(sources, SourceRef{
			ChunkID:   c.ID,
			Title:     c.Title,
			Provider:  c.Provider,
			SourceURL: c.SourceURL,
		})
	}

	return &AskResponse{
		Answer:   answer,
		Provider: provider,
		Sources:  sources,
	}, nil
}

func detectLang(s string) string {
    info := wl.Detect(s)
    switch wl.LangToString(info.Lang) {
    case "Por": return "pt"
    case "Eng": return "en"
    case "Spa": return "es"
    default:
        return "pt"
    }
}

func resolveProvider(p *Provider, question string) Provider {
	if p != nil && *p != "" {
		return *p
	}

	q := strings.ToLower(question)

	switch {
	case strings.Contains(q, "entrepay"):
		return ProviderEntrepay
	case strings.Contains(q, "e-rede"),
		strings.Contains(q, "erede"),
		strings.Contains(q, "e rede"),
		strings.Contains(q, "rede"):
		return ProviderRede
	default:
		return ""
	}
}
