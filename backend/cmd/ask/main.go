package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

func main() {
	question := flag.String("q", "", "pertanyaan pelanggan")
	siteKey := flag.String("site", "demo-kopi", "site_key bisnis")
	topK := flag.Int("k", 3, "jumlah chunk yang diambil")
	flag.Parse()
	if *question == "" {
		log.Fatal(`pakai: go run ./cmd/ask -q "tutup jam berapa?"`)
	}

	dbURL := os.Getenv("DATABASE_URL")
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_EMBED_MODEL")
	if dbURL == "" || apiKey == "" || model == "" {
		log.Fatal("DATABASE_URL, GEMINI_API_KEY, GEMINI_EMBED_MODEL wajib diisi (jalankan: set -a; source .env; set +a)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Ubah pertanyaan jadi sidik jari (pakai model yang SAMA dengan saat ingest)
	var provider llm.LLM = llm.NewGeminiProvider(apiKey, model, 768)
	qvec, err := provider.Embed(ctx, *question)
	if err != nil {
		log.Fatalf("embed pertanyaan: %v", err)
	}

	// 2. Koneksi database
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("koneksi db: %v", err)
	}
	defer conn.Close(ctx)
	if err := pgxvec.RegisterTypes(ctx, conn); err != nil {
		log.Fatalf("register pgvector: %v", err)
	}

	// 3. Cari chunk paling mirip, khusus milik bisnis ini
	rows, err := conn.Query(ctx, `
		SELECT d.title, c.chunk_index, c.content, 1 - (c.embedding <=> $1) AS similarity
		FROM chunks c
		JOIN documents d  ON d.id = c.document_id
		JOIN businesses b ON b.id = c.business_id
		WHERE b.site_key = $2
		ORDER BY c.embedding <=> $1
		LIMIT $3`,
		pgvector.NewVector(qvec), *siteKey, *topK,
	)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	// 4. Tampilkan hasilnya
	fmt.Printf("Pertanyaan: %s\n\n", *question)
	for rows.Next() {
		var title, content string
		var idx int
		var similarity float64
		if err := rows.Scan(&title, &idx, &content, &similarity); err != nil {
			log.Fatalf("scan: %v", err)
		}
		firstLine := strings.SplitN(content, "\n", 2)[0]
		fmt.Printf("[%.3f] %s #%d: %s\n", similarity, title, idx, firstLine)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}
}

