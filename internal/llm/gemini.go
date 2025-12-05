package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/josinaldojr/payment-gateway-rag/internal/rag"
	"google.golang.org/genai"
)

const (
	embeddingModel = "models/text-embedding-004"
	ragChatModel   = "gemini-2.5-flash"
	embedDim       = 768
)

type GeminiClient struct {
	client *genai.Client
}

func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("missing GOOGLE_API_KEY or GEMINI_API_KEY")
	}

	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &GeminiClient{client: c}, nil
}

func (g *GeminiClient) Embed(ctx context.Context, text string) ([]float32, error) {
	clean := normalizeWhitespace(text)
	if clean == "" {
		return nil, fmt.Errorf("empty text for embedding")
	}

	resp, err := g.client.Models.EmbedContent(
		ctx,
		embeddingModel,
		genai.Text(clean),
		&genai.EmbedContentConfig{
			OutputDimensionality: genai.Ptr(int32(embedDim)),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini embed error: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	values := resp.Embeddings[0].Values
	if len(values) != embedDim {
		return nil, fmt.Errorf("unexpected embedding size %d (expected %d)", len(values), embedDim)
	}

	out := make([]float32, embedDim)
	for i, v := range values {
		out[i] = float32(v)
	}
	return out, nil
}

func (g *GeminiClient) GenerateAnswer(
	ctx context.Context,
	question string,
	chunks []rag.DocChunk,
	provider rag.Provider,
	lang string,
) (string, error) {
	if len(chunks) == 0 {
		return "I couldn't find any relevant information in the indexed documentation for this question.", nil
	}

	systemPrompt, contextText := buildSystemPrompt(provider, chunks, lang)

	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.Text(systemPrompt)[0],
	}

	userContent := fmt.Sprintf(
		"Question:\n%s\n\nRelevant documentation excerpts:\n%s",
		strings.TrimSpace(question),
		contextText,
	)

	resp, err := g.client.Models.GenerateContent(
		ctx,
		ragChatModel,
		genai.Text(userContent),
		cfg,
	)
	if err != nil {
		return "", fmt.Errorf("gemini generateContent error: %w", err)
	}

	if resp == nil {
		return "", fmt.Errorf("empty response from gemini")
	}

	txt := strings.TrimSpace(resp.Text())
	if txt == "" {
		return "", fmt.Errorf("model returned empty text")
	}

	return txt, nil
}

// -------- helpers --------

func buildSystemPrompt(provider rag.Provider, chunks []rag.DocChunk, lang string) (string, string) {
	var sys strings.Builder
	var ctx strings.Builder

	target := map[string]string{
		"pt": "Brazilian Portuguese",
		"en": "English",
		"es": "Spanish",
	}[lang]
	if target == "" {
		target = "Brazilian Portuguese"
	}

	sys.WriteString("You are a technical assistant specialized in payment gateway integrations for ")
	sys.WriteString(string(provider))
	sys.WriteString(". ")
	sys.WriteString(target)
	sys.WriteString(" is the target language for all responses. ")
	sys.WriteString("Always answer ONLY based on the provided documentation excerpts. ")
	sys.WriteString("If the answer is not clearly present, say that it is not available in the indexed documentation. ")
	sys.WriteString("Do not invent endpoints, URLs, fields or values. ")
	sys.WriteString("When possible, structure the answer as:\n")
	sys.WriteString("- Operation flow\n")
	sys.WriteString("- Endpoint(s)\n")
	sys.WriteString("- Required and optional parameters\n")
	sys.WriteString("- Example request/response\n")
	sys.WriteString("- Important notes (3DS, capture, refunds, error codes, etc.)\n")

	const (
		maxChunks     = 10
		maxChunkChars = 1200
	)

	n := len(chunks)
	if n > maxChunks {
		n = maxChunks
	}

	for i := 0; i < n; i++ {
		c := chunks[i]
		ctx.WriteString(fmt.Sprintf(
			"\n[DOC %d] title=%s source=%s\n",
			c.ID,
			oneLine(c.Title),
			c.SourceURL,
		))
		ctx.WriteString(trimBody(c.Content, maxChunkChars))
		ctx.WriteString("\n----\n")
	}

	return sys.String(), ctx.String()
}

func normalizeWhitespace(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			if !space {
				b.WriteRune(' ')
				space = true
			}
		} else {
			b.WriteRune(r)
			space = false
		}
	}
	return b.String()
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "..."
	}
	return s
}

func trimBody(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

var _ rag.EmbeddingsClient = (*GeminiClient)(nil)
var _ rag.LLMClient = (*GeminiClient)(nil)
