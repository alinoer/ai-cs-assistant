package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/chunk"
	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

const maxChunkChars = 2000 // kira-kira 500 token

func main() {
	file := flag.String("file", "", "path ke file .md")
	siteKey := flag.String("site", "demo-kopi", "site_key bisnis")
	flag.Parse()
	if *file == "" {
		log.Fatal("pakai: go run ./cmd/ingest -file data/kedai-kopi.md")
	}

	dbURL := os.Getenv("DATABASE_URL")
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_EMBED_MODEL")
	if dbURL == "" || apiKey == "" || model == "" {
		log.Fatal("DATABASE_URL, GEMINI_API_KEY, GEMINI_EMBED_MODEL wajib diisi (jalankan: set -a; source .env; set +a)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Baca file
	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("baca file: %v", err)
	}

	// 2. Potong jadi chunk, lalu embed satu per satu (sebelum buka transaksi)
	chunks := chunk.Split(string(raw), maxChunkChars)
	if len(chunks) == 0 {
		log.Fatal("dokumen kosong, tidak ada yang di-ingest")
	}

	var provider llm.LLM = llm.NewGeminiProvider(apiKey, model, 768)
	vectors := make([][]float32, len(chunks))
	for i, c := range chunks {
		vec, err := provider.Embed(ctx, c.Content)
		if err != nil {
			log.Fatalf("embed chunk %d (%s): %v", i, c.Heading, err)
		}
		vectors[i] = vec
		fmt.Printf("  embed %d/%d: %s\n", i+1, len(chunks), c.Heading)
	}

	// 3. Koneksi ke database + daftarkan tipe vector
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("koneksi db: %v", err)
	}
	defer conn.Close(ctx)

	if err := pgxvec.RegisterTypes(ctx, conn); err != nil {
		log.Fatalf("register pgvector: %v", err)
	}

	// 4. Simpan dalam satu transaksi: 1 document + N chunk
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	var businessID int64
	if err := tx.QueryRow(ctx,
		`SELECT id FROM businesses WHERE site_key = $1`, *siteKey,
	).Scan(&businessID); err != nil {
		log.Fatalf("cari bisnis %q: %v", *siteKey, err)
	}

	title := filepath.Base(*file)
	var documentID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO documents (business_id, title) VALUES ($1, $2) RETURNING id`,
		businessID, title,
	).Scan(&documentID); err != nil {
		log.Fatalf("insert document: %v", err)
	}

	for i, c := range chunks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO chunks (document_id, business_id, chunk_index, content, embedding)
			 VALUES ($1, $2, $3, $4, $5)`,
			documentID, businessID, i, c.Content, pgvector.NewVector(vectors[i]),
		); err != nil {
			log.Fatalf("insert chunk %d: %v", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Printf("OK: document %d (%s), %d chunk tersimpan\n", documentID, title, len(chunks))
}
