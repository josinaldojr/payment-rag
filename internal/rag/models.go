package rag

import "time"

// Provider identifica de qual gateway vem a doc.
// Já deixo tipado p/ evitar string solta no código.
type Provider string

const (
	ProviderRede     Provider = "rede"
	ProviderEntrepay Provider = "entrepay"
	// Futuro: ProviderCielo, ProviderStripe...
)

type SectionType string

const (
	SectionOverview SectionType = "overview"
	SectionAuth     SectionType = "auth"
	SectionEndpoint SectionType = "endpoint"
	Section3DS      SectionType = "3ds"
	SectionErrors   SectionType = "errors"
)

// DocChunk
// Um pedaço lógico da documentação (um endpoint, uma seção 3DS, uma tabela de códigos etc).
type DocChunk struct {
	ID          int64       `json:"id"`
	Provider    Provider    `json:"provider"`
	SectionType SectionType `json:"sectionType"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	SourceURL   string      `json:"sourceUrl"`
	APIVersion  string      `json:"apiVersion"`
	Tags        []string    `json:"tags"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// DocChunkEmbedding
// Vetor associado a um chunk, usando pgvector no Postgres.
type DocChunkEmbedding struct {
	ChunkID   int64     `json:"chunkId"`
	Embedding []float32 `json:"embedding"`
	CreatedAt time.Time `json:"createdAt"`
}

// AskRequest
// Payload da sua API /ask.
type AskRequest struct {
	Question string    `json:"question"`
	Provider *Provider `json:"provider,omitempty"` // opcional; se vazio, você detecta pelo texto
	TopK     int       `json:"topK,omitempty"`     // opcional; default interno
	Lang     string      `json:"lang"`
}

// SourceRef
// Metadados dos trechos usados para montar a resposta.
type SourceRef struct {
	ChunkID   int64    `json:"chunkId"`
	Title     string   `json:"title"`
	Provider  Provider `json:"provider"`
	SourceURL string   `json:"sourceUrl"`
}

// AskResponse
// Resposta da API: texto + fontes.
type AskResponse struct {
	Answer   string      `json:"answer"`
	Provider Provider    `json:"provider"`
	Sources  []SourceRef `json:"sources"`
}
