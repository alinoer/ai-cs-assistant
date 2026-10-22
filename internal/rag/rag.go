package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
)

const notFoundAnswer = "Maaf, informasi tersebut belum tersedia. Silakan hubungi kami langsung untuk bantuan lebih lanjut."

const systemPrompt = `Kamu adalah asisten customer service untuk sebuah bisnis.
Aturan:
1. Jawab HANYA berdasarkan KONTEKS yang diberikan. Jangan menebak dan jangan memakai pengetahuan umum.
2. Kalau jawabannya tidak ada di KONTEKS: set "answered" ke false, "sources" kosong, dan "answer" berisi permintaan maaf singkat bahwa informasinya belum tersedia.
3. Kalau jawabannya ada: set "answered" ke true, dan isi "sources" dengan nomor konteks yang kamu pakai, misalnya [1].
4. Jawab singkat, ramah, dan dalam bahasa yang sama dengan pertanyaan pelanggan.`

var answerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answer":   map[string]any{"type": "string"},
		"answered": map[string]any{"type": "boolean"},
		"sources": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "integer"},
		},
	},
	"required":             []string{"answer", "answered", "sources"},
	"additionalProperties": false,
}

// ErrEmptyQuestion dikembalikan kalau pertanyaan kosong (nanti jadi HTTP 400).
var ErrEmptyQuestion = errors.New("pertanyaan kosong")

// Source adalah potongan dokumen yang dipakai untuk menjawab.
type Source struct {
	Title      string  `json:"title"`
	Section    string  `json:"section"`
	Similarity float64 `json:"similarity"`
}

// Result adalah hasil akhir yang dikirim ke pelanggan.
type Result struct {
	Answer   string   `json:"answer"`
	Answered bool     `json:"answered"`
	Sources  []Source `json:"sources"`
}

type Service struct {
	db            *pgxpool.Pool
	embedder      llm.Embedder
	generator     llm.Generator
	topK          int
	minSimilarity float64
}

func NewService(db *pgxpool.Pool, embedder llm.Embedder, generator llm.Generator) *Service {
	return &Service{
		db:            db,
		embedder:      embedder,
		generator:     generator,
		topK:          3,
		minSimilarity: 0.50, // saringan kasar; keputusan akhir di LLM
	}
}

type retrievedChunk struct {
	title      string
	content    string
	similarity float64
}

func (s *Service) Answer(ctx context.Context, siteKey, question string) (Result, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return Result{}, ErrEmptyQuestion
	}

	// 1. Ubah pertanyaan jadi sidik jari
	qvec, err := s.embedder.Embed(ctx, question)
	if err != nil {
		return Result{}, fmt.Errorf("embed pertanyaan: %w", err)
	}

	// 2. Ambil chunk paling mirip milik bisnis ini
	chunks, err := s.retrieve(ctx, siteKey, qvec)
	if err != nil {
		return Result{}, err
	}

	// 3. Saringan kasar: buang yang jelas tidak nyambung
	var relevant []retrievedChunk
	for _, c := range chunks {
		if c.similarity >= s.minSimilarity {
			relevant = append(relevant, c)
		}
	}
	if len(relevant) == 0 {
		return notFound(), nil // hemat: tidak perlu memanggil LLM
	}

	// 4. Minta LLM menjawab berdasarkan chunk
	raw, err := s.generator.Generate(ctx, llm.GenerateRequest{
		System:     systemPrompt,
		Prompt:     buildPrompt(question, relevant),
		SchemaName: "cs_answer",
		JSONSchema: answerSchema,
	})
	if err != nil {
		return Result{}, fmt.Errorf("generate jawaban: %w", err)
	}

	// 5. Baca & validasi jawaban LLM
	var out struct {
		Answer   string `json:"answer"`
		Answered bool   `json:"answered"`
		Sources  []int  `json:"sources"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return Result{}, fmt.Errorf("parse jawaban: %w", err)
	}
	if strings.TrimSpace(out.Answer) == "" {
		return Result{}, errors.New("jawaban LLM kosong")
	}
	if !out.Answered {
		return Result{Answer: out.Answer, Answered: false, Sources: []Source{}}, nil
	}

	// 6. Ubah nomor sumber [1],[2] jadi data sumber yang jelas
	result := Result{Answer: out.Answer, Answered: true, Sources: []Source{}}
	seen := map[int]bool{}
	for _, n := range out.Sources {
		if n < 1 || n > len(relevant) || seen[n] {
			continue // abaikan nomor yang tidak ada atau dobel
		}
		seen[n] = true
		c := relevant[n-1]
		result.Sources = append(result.Sources, Source{
			Title:      c.title,
			Section:    firstLine(c.content),
			Similarity: c.similarity,
		})
	}

	// Jawaban "terjawab" tanpa sumber yang valid = mencurigakan → anggap tidak terjawab
	if len(result.Sources) == 0 {
		return notFound(), nil
	}
	return result, nil
}

func (s *Service) retrieve(ctx context.Context, siteKey string, qvec []float32) ([]retrievedChunk, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.title, c.content, 1 - (c.embedding <=> $1) AS similarity
		FROM chunks c
		JOIN documents d  ON d.id = c.document_id
		JOIN businesses b ON b.id = c.business_id
		WHERE b.site_key = $2
		ORDER BY c.embedding <=> $1
		LIMIT $3`,
		pgvector.NewVector(qvec), siteKey, s.topK,
	)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}
	defer rows.Close()

	var out []retrievedChunk
	for rows.Next() {
		var c retrievedChunk
		if err := rows.Scan(&c.title, &c.content, &c.similarity); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func buildPrompt(question string, chunks []retrievedChunk) string {
	var b strings.Builder
	b.WriteString("KONTEKS:\n")
	for i, c := range chunks {
		fmt.Fprintf(&b, "[%d]\n%s\n\n", i+1, c.content)
	}
	fmt.Fprintf(&b, "PERTANYAAN PELANGGAN:\n%s", question)
	return b.String()
}

func firstLine(s string) string {
	return strings.SplitN(s, "\n", 2)[0]
}

func notFound() Result {
	return Result{Answer: notFoundAnswer, Answered: false, Sources: []Source{}}
}

