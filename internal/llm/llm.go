package llm

import "context"

// GenerateRequest berisi semua yang dikirim saat meminta jawaban ke LLM.
type GenerateRequest struct {
	System     string         // aturan & peran untuk LLM
	Prompt     string         // pertanyaan (+ konteks chunk)
	SchemaName string         // nama skema, wajib untuk json_schema (mis. "cs_answer")
	JSONSchema map[string]any // JSON Schema standar; kalau diisi, jawaban WAJIB JSON
}

// Embedder: bisa mengubah teks jadi sidik jari (vector).
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Generator: bisa membuat jawaban teks/JSON.
type Generator interface {
	Generate(ctx context.Context, req GenerateRequest) (string, error)
}

// LLM: provider yang bisa keduanya (mis. Gemini).
type LLM interface {
	Embedder
	Generator
}
