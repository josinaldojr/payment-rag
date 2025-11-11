# 🧠 Payment Gateway RAG Assistant

Um sistema **RAG (Retrieval-Augmented Generation)** desenvolvido em **Go (Golang)** para ajudar engenheiros de software a compreender e integrar **gateways de pagamento**, como **e-Rede** e **Entrepay**, respondendo perguntas técnicas com base em suas documentações oficiais.

---

## 🚀 Funcionalidades

- 📘 **Importação de documentação**:
  - PDF, HTML, Markdown e texto simples.
  - Suporte para docs offline ou via web crawler.
- 🧩 **Vetorização com Gemini Embeddings** (`models/text-embedding-004` – 768 dimensões).
- 💬 **Geração de respostas** com contexto técnico (modelo `gemini-2.5-flash`).
- 🗄️ **Armazenamento vetorial** no PostgreSQL usando `pgvector`.
- 🧠 **Busca semântica** e contexto otimizado (chunking de 2000 caracteres com limpeza UTF-8).
- 🔍 **Endpoint `/ask`**: consulta natural à documentação indexada.

---

## 🧰 Stack técnica

| Componente | Descrição |
|-----------|-----------|
| **Golang 1.22+** | Linguagem principal |
| **PostgreSQL + pgvector** | Banco de dados vetorial |
| **Gemini API (Google)** | Embeddings e geração de texto |
| **Docker Compose** | Infraestrutura local |
| **REST API** | Interface de consulta HTTP |

---

## 🧑‍💻 Estrutura de diretórios

```text
payment-gateway-rag/
├── cmd/
│   ├── api/                # API HTTP /ask
│   │   └── main.go
│   └── import-doc/         # Importador de documentação (PDF/HTML/TXT)
│       └── main.go
├── internal/
│   ├── config/             # Configurações do ambiente
│   ├── db/                 # Conexão com PostgreSQL
│   ├── llm/                # Cliente Gemini (Embed + GenerateAnswer)
│   ├── rag/                # Lógica principal de RAG (repositório, serviço)
│   └── http/               # Handlers e rotas REST
├── docs/
│   └── rede/               # PDFs e docs locais (e-Rede, etc.)
├── go.mod
└── README.md
```

---

## ⚙️ Configuração

### 1. Variáveis de ambiente

Crie um arquivo `.env` na raiz:

```bash
DATABASE_URL=postgres://rag_user:rag_pass@localhost:5432/payment_gateway_rag?sslmode=disable
GOOGLE_API_KEY=your_gemini_api_key_here
PORT=8080
```

### 2. Banco de dados

Certifique-se de ter o PostgreSQL rodando com a extensão `pgvector`.

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### 3. Docker Compose (opcional)

```yaml
version: '3.8'
services:
  db:
    image: ankane/pgvector
    environment:
      POSTGRES_USER: rag_user
      POSTGRES_PASSWORD: rag_pass
      POSTGRES_DB: payment_gateway_rag
    ports:
      - "5432:5432"
```

---

## 🧩 Importando documentação

### Opção A: via arquivo local (PDF, TXT, HTML, MD)

1. Baixe a documentação desejada (ex: `e-rede_26102025.pdf`).
2. Salve em `./docs/rede/`.
3. Rode:

```bash
go run ./cmd/import-doc   --provider=rede   --from-files   --path=./docs/rede
```

O importador:

- Lê `.pdf`, `.html`, `.md`, `.txt`.
- Extrai o texto.
- Limpa caracteres inválidos (UTF-8).
- Quebra em chunks de até 2000 caracteres.
- Gera embeddings com Gemini.
- Salva em `doc_chunk` + `doc_chunk_embedding`.

### Opção B: via URL (crawler simples)

```bash
go run ./cmd/import-doc   --provider=rede   --from-url   --base-url=https://developer.userede.com.br/e-rede   --max-pages=40
```

> ⚠️ Documentações SPA (como o portal da Rede) podem não renderizar completamente via HTTP simples. Prefira o PDF ou exportações estáticas.

---

## 🧠 Consultando via API

### 1. Iniciar a API

```bash
go run ./cmd/api
```

Saída esperada:

```text
API listening on :8080
```

### 2. Endpoint `/ask`

**Request**:

```http
POST /ask
Content-Type: application/json
```

```json
{
  "question": "How do I create a 3DS transaction with e-Rede? Show endpoint and required fields.",
  "provider": "rede",
  "topK": 8
}
```

**Response (exemplo)**:

```json
{
  "answer": "To create a 3DS transaction with e-Rede, use the POST /... endpoint with the following required fields: ...",
  "provider": "rede",
  "sources": [
    {
      "chunkId": 45,
      "title": "e-rede_26102025 (part 23)",
      "sourceUrl": ""
    }
  ]
}
```

A resposta:

- É sempre baseada apenas nos trechos indexados.
- Inclui `sources` para rastrear de qual parte da documentação veio.

---

## 🧹 Limpar e reimportar documentos

Para resetar a base de um provider (ex: `rede`):

```sql
DELETE FROM doc_chunk_embedding
WHERE chunk_id IN (SELECT id FROM doc_chunk WHERE provider = 'rede');

DELETE FROM doc_chunk
WHERE provider = 'rede';
```

Reimportar:

```bash
go run ./cmd/import-doc   --provider=rede   --from-files   --path=./docs/rede
```

---

## ⚡ Solução de problemas

### `ERROR: expected 768 dimensions, not 3072`

- Ajustar o `GeminiClient.Embed` para usar `OutputDimensionality = 768`.
- Garantir que a coluna é `VECTOR(768)`.

### `ERROR: invalid byte sequence for encoding "UTF8"`

- Já tratado com `sanitizeUTF8` no importador.
- Certifica que está usando a versão atual de `cmd/import-doc/main.go`.

### `Error 504 DEADLINE_EXCEEDED` (Gemini)

- Reduzido com:
  - limite de chunks (`maxChunks = 10`);
  - limite por chunk no prompt (`maxChunkChars = 1200`);
  - prompt de sistema enxuto.

### Resposta: `"I could not find information..."`

- Verifique se:
  - A documentação correta foi importada (PDF/TXT certo).
  - Existem chunks com o termo consultado:
    ```sql
    SELECT COUNT(*) FROM doc_chunk
    WHERE provider = 'rede'
      AND content ILIKE '%3DS%';
    ```

---

## 🧠 Extensões futuras

- Suporte a múltiplos gateways (Entrepay, Cielo, Stone, etc.) usando `provider`.
- UI web (Next.js) consumindo `/ask`.
- Autenticação e controle de acesso.
- Cache para respostas frequentes.
- Reindexação automática via pipeline CI/CD.

---

## 📄 Licença

MIT — use, adapte e melhore.