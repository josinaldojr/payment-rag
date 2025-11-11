package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	pdf "github.com/dslipak/pdf"
	"github.com/joho/godotenv"
	"github.com/josinaldojr/payment-gateway-rag/internal/config"
	"github.com/josinaldojr/payment-gateway-rag/internal/db"
	"github.com/josinaldojr/payment-gateway-rag/internal/llm"
	"github.com/josinaldojr/payment-gateway-rag/internal/rag"
	"golang.org/x/net/html"
)

func main() {
	_ = godotenv.Load()

	providerFlag := flag.String("provider", "", "gateway provider (ex: rede, entrepay)")
	fromFiles := flag.Bool("from-files", false, "importar a partir de arquivos locais (.md/.txt/.html/.pdf)")
	pathFlag := flag.String("path", "", "diretório base para arquivos locais")
	fromURL := flag.Bool("from-url", false, "importar via crawl HTTP")
	baseURLFlag := flag.String("base-url", "", "URL base para crawl (ex: https://developer.userede.com.br/e-rede)")
	maxPagesFlag := flag.Int("max-pages", 50, "limite de páginas para crawl HTTP")
	apiVersionFlag := flag.String("api-version", "", "versão da API (opcional)")
	flag.Parse()

	if *providerFlag == "" {
		log.Fatal("obrigatório: --provider")
	}
	provider := rag.Provider(*providerFlag)

	if !*fromFiles && !*fromURL {
		log.Fatal("use pelo menos um modo: --from-files ou --from-url")
	}

	ctx := context.Background()
	cfg := config.Load()
	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	repo := rag.NewPgRepository(pool)

	geminiClient, err := llm.NewGeminiClient(ctx)
	if err != nil {
		log.Fatalf("erro ao iniciar Gemini: %v", err)
	}

	if *fromFiles {
		if *pathFlag == "" {
			log.Fatal("--path é obrigatório com --from-files")
		}
		if err := importFromFiles(ctx, repo, geminiClient, provider, *pathFlag, *apiVersionFlag); err != nil {
			log.Fatalf("erro importando arquivos: %v", err)
		}
	}

	if *fromURL {
		if *baseURLFlag == "" {
			log.Fatal("--base-url é obrigatório com --from-url")
		}
		if err := importFromHTTP(ctx, repo, geminiClient, provider, *baseURLFlag, *apiVersionFlag, *maxPagesFlag); err != nil {
			log.Fatalf("erro importando HTTP: %v", err)
		}
	}

	log.Println("✅ Importação concluída.")
}

func importFromFiles(
	ctx context.Context,
	repo *rag.PgRepository,
	gemini *llm.GeminiClient,
	provider rag.Provider,
	rootPath string,
	apiVersion string,
) error {
	log.Printf("📂 Importando docs locais de %s para provider=%s", rootPath, provider)

	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isTextFile(path) {
			return nil
		}

		lpath := strings.ToLower(path)
		var content string

		switch {
		case strings.HasSuffix(lpath, ".pdf"):
			text, err := extractTextFromPDF(path)
			if err != nil {
				return fmt.Errorf("erro lendo pdf %s: %w", path, err)
			}
			content = text

		case strings.HasSuffix(lpath, ".html") || strings.HasSuffix(lpath, ".htm"):
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("erro lendo %s: %w", path, err)
			}
			content = extractMainText(string(data))

		default: 
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("erro lendo %s: %w", path, err)
			}
			content = string(data)
		}

		content = strings.TrimSpace(content)
		content = sanitizeUTF8(content)
		if content == "" {
			return nil
		}

		title := filenameToTitle(path)
		return chunkAndStore(ctx, repo, gemini, provider, title, "", apiVersion, content)
	})
}

func importFromHTTP(
	ctx context.Context,
	repo *rag.PgRepository,
	gemini *llm.GeminiClient,
	provider rag.Provider,
	baseURL, apiVersion string,
	maxPages int,
) error {
	log.Printf("🌐 Crawl HTTP: base=%s provider=%s maxPages=%d", baseURL, provider, maxPages)

	base, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("base-url inválida: %w", err)
	}

	visited := make(map[string]bool)
	queue := []string{base.String()}
	pages := 0

	for len(queue) > 0 && pages < maxPages {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		pages++

		log.Printf("Baixando %s", current)
		resp, err := http.Get(current)
		if err != nil {
			log.Printf("erro GET %s: %v", current, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("status %d em %s", resp.StatusCode, current)
			resp.Body.Close()
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("erro lendo body %s: %v", current, err)
			continue
		}

		htmlStr := string(bodyBytes)
		text := extractMainText(htmlStr)
		text = strings.TrimSpace(text)
		text = sanitizeUTF8(text)
		if text != "" {
			title := urlToTitle(current, base)
			if err := chunkAndStore(ctx, repo, gemini, provider, title, current, apiVersion, text); err != nil {
				log.Printf("erro salvando chunks de %s: %v", current, err)
			}
		}

		for _, link := range extractLinks(htmlStr, base) {
			if !visited[link] {
				queue = append(queue, link)
			}
		}
	}

	return nil
}

func isTextFile(path string) bool {
	l := strings.ToLower(path)
	return strings.HasSuffix(l, ".md") ||
		strings.HasSuffix(l, ".txt") ||
		strings.HasSuffix(l, ".html") ||
		strings.HasSuffix(l, ".htm") ||
		strings.HasSuffix(l, ".pdf")
}

func filenameToTitle(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ReplaceAll(base, "-", " ")
	return strings.TrimSpace(base)
}

func urlToTitle(raw string, base *url.URL) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Path == base.Path || u.Path == base.Path+"/" {
		return "Overview"
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	last := parts[len(parts)-1]
	last = strings.SplitN(last, ".", 2)[0]
	last = strings.ReplaceAll(last, "-", " ")
	return strings.TrimSpace(last)
}

func extractMainText(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	var b strings.Builder
	var walk func(*html.Node, bool)

	walk = func(n *html.Node, skip bool) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				skip = true
			}
		}

		if n.Type == html.TextNode && !skip {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				b.WriteString(t)
				b.WriteString("\n")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, skip)
		}
	}
	walk(doc, false)

	lines := strings.Split(b.String(), "\n")
	var filtered []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && len(l) > 1 {
			filtered = append(filtered, l)
		}
	}
	return strings.Join(filtered, "\n")
}

func extractLinks(htmlStr string, base *url.URL) []string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}
	var links []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					h := strings.TrimSpace(a.Val)
					if h == "" || strings.HasPrefix(h, "#") {
						continue
					}
					u, err := url.Parse(h)
					if err != nil {
						continue
					}
					u = base.ResolveReference(u)

					if u.Host != base.Host {
						continue
					}

					if strings.HasSuffix(u.Path, ".css") ||
						strings.HasSuffix(u.Path, ".js") ||
						strings.HasSuffix(u.Path, ".png") ||
						strings.HasSuffix(u.Path, ".jpg") ||
						strings.HasSuffix(u.Path, ".svg") {
						continue
					}

					link := u.Scheme + "://" + u.Host + u.Path
					links = append(links, link)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	seen := make(map[string]bool)
	var out []string
	for _, l := range links {
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}

func chunkAndStore(
	ctx context.Context,
	repo *rag.PgRepository,
	gemini *llm.GeminiClient,
	provider rag.Provider,
	title, sourceURL, apiVersion, content string,
) error {
	const maxLen = 2000

	chunks := splitIntoChunks(content, maxLen)
	if len(chunks) == 0 {
		return nil
	}

	for i, c := range chunks {
		c = strings.TrimSpace(c)
		c = sanitizeUTF8(c)
		if c == "" {
			continue
		}

		chunkTitle := title
		if len(chunks) > 1 {
			chunkTitle = fmt.Sprintf("%s (parte %d)", title, i+1)
		}

		doc := &rag.DocChunk{
			Provider:    provider,
			SectionType: detectSectionType(c),
			Title:       chunkTitle,
			Content:     c,
			SourceURL:   sourceURL,
			APIVersion:  apiVersion,
			Tags:        detectTags(c),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		vec, err := gemini.Embed(ctx, c)
		if err != nil {
			return fmt.Errorf("embedding error: %w", err)
		}

		id, err := repo.InsertChunk(ctx, doc, vec)
		if err != nil {
			return fmt.Errorf("insert chunk error: %w", err)
		}

		log.Printf("✅ chunk importado provider=%s id=%d len=%d title=%s", provider, id, len(c), chunkTitle)
	}

	return nil
}

func splitIntoChunks(content string, maxLen int) []string {
	content = strings.TrimSpace(content)
	content = sanitizeUTF8(content)
	if content == "" {
		return nil
	}
	if len(content) <= maxLen {
		return []string{content}
	}

	var chunks []string
	var buf strings.Builder

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		chunk := strings.TrimSpace(buf.String())
		chunk = sanitizeUTF8(chunk)
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		buf.Reset()
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		for len(line) > maxLen {
			part := line[:maxLen]
			line = line[maxLen:]

			if buf.Len() > 0 {
				flush()
			}
			buf.WriteString(part)
			flush()
		}

		if buf.Len()+len(line)+1 > maxLen {
			flush()
		}

		buf.WriteString(line)
		buf.WriteRune('\n')
	}

	flush()
	return chunks
}

func detectSectionType(chunk string) rag.SectionType {
	s := strings.ToLower(chunk)

	switch {
	case strings.Contains(s, "3ds") || strings.Contains(s, "3-d secure"):
		return rag.Section3DS
	case strings.Contains(s, "authorization") || strings.Contains(s, "autorização"):
		return rag.SectionAuth
	case strings.Contains(s, "endpoint") ||
		(strings.Contains(s, "http") &&
			(strings.Contains(s, "post") || strings.Contains(s, "get") ||
				strings.Contains(s, "put") || strings.Contains(s, "delete"))):
		return rag.SectionEndpoint
	case strings.Contains(s, "error code") || strings.Contains(s, "código de erro"):
		return rag.SectionErrors
	default:
		return rag.SectionOverview
	}
}

func detectTags(chunk string) []string {
	s := strings.ToLower(chunk)
	var tags []string

	add := func(t string) {
		for _, ex := range tags {
			if ex == t {
				return
			}
		}
		tags = append(tags, t)
	}

	if strings.Contains(s, "3ds") || strings.Contains(s, "3-d secure") {
		add("3ds")
		add("auth")
	}
	if strings.Contains(s, "authorization") || strings.Contains(s, "autorização") {
		add("authorization")
	}
	if strings.Contains(s, "capture") || strings.Contains(s, "captura") {
		add("capture")
	}
	if strings.Contains(s, "refund") || strings.Contains(s, "estorno") {
		add("refund")
	}
	if strings.Contains(s, "cancel") || strings.Contains(s, "void") {
		add("cancel")
	}
	if strings.Contains(s, "webhook") || strings.Contains(s, "notificação") {
		add("webhook")
	}
	if strings.Contains(s, "sandbox") {
		add("sandbox")
	}
	if strings.Contains(s, "transaction") || strings.Contains(s, "transação") {
		add("transaction")
	}

	return tags
}

func extractTextFromPDF(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}

	reader, err := r.GetPlainText()
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", err
	}

	text := strings.TrimSpace(buf.String())
	text = sanitizeUTF8(text)
	return text, nil
}

// remove bytes inválidos para UTF-8 (evita erro 22021 no Postgres)
func sanitizeUTF8(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			// byte inválido: descarta
			s = s[1:]
			continue
		}
		b.WriteRune(r)
		s = s[size:]
	}
	return b.String()
}
