package rag

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type Repository interface {
	InsertChunk(ctx context.Context, c *DocChunk, embedding []float32) (int64, error)
	GetChunksByIDs(ctx context.Context, ids []int64) ([]DocChunk, error)
	SearchSimilarChunks(ctx context.Context, provider Provider, embedding []float32, limit int) ([]DocChunk, error)
}

type PgRepository struct {
	db *pgxpool.Pool
}

func NewPgRepository(db *pgxpool.Pool) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) InsertChunk(ctx context.Context, c *DocChunk, embedding []float32) (int64, error) {
	var id int64

	err := r.db.QueryRow(ctx, `
		INSERT INTO doc_chunk (provider, section_type, title, content, source_url, api_version, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		c.Provider,
		c.SectionType,
		c.Title,
		c.Content,
		c.SourceURL,
		c.APIVersion,
		c.Tags,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if embedding != nil {
		vec := pgvector.NewVector(embedding)
		_, err = r.db.Exec(ctx, `
			INSERT INTO doc_chunk_embedding (chunk_id, embedding)
			VALUES ($1, $2)
		`, id, vec)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (r *PgRepository) GetChunksByIDs(ctx context.Context, ids []int64) ([]DocChunk, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, provider, section_type, title, content, source_url, api_version, tags, created_at, updated_at
		FROM doc_chunk
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []DocChunk
	for rows.Next() {
		var c DocChunk
		if err := rows.Scan(
			&c.ID,
			&c.Provider,
			&c.SectionType,
			&c.Title,
			&c.Content,
			&c.SourceURL,
			&c.APIVersion,
			&c.Tags,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// SearchSimilarChunks faz a busca vetorial filtrando por provider.
func (r *PgRepository) SearchSimilarChunks(ctx context.Context, provider Provider, embedding []float32, limit int) ([]DocChunk, error) {
	if limit <= 0 {
		limit = 5
	}

	vec := pgvector.NewVector(embedding)

	rows, err := r.db.Query(ctx, `
		SELECT 
			c.id, c.provider, c.section_type, c.title, c.content,
			c.source_url, c.api_version, c.tags, c.created_at, c.updated_at
		FROM doc_chunk c
		JOIN doc_chunk_embedding e ON c.id = e.chunk_id
		WHERE c.provider = $1
		ORDER BY e.embedding <-> $2
		LIMIT $3
	`, provider, vec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []DocChunk
	for rows.Next() {
		var c DocChunk
		if err := rows.Scan(
			&c.ID,
			&c.Provider,
			&c.SectionType,
			&c.Title,
			&c.Content,
			&c.SourceURL,
			&c.APIVersion,
			&c.Tags,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}
