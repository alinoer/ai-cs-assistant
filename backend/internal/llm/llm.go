package llm

import "context"

// GenerateRequest berisi semua yang dikirim saat meminta jawaban ke LLM.
type GenerateRequest struct {
	System     string         // aturan & peran untuk LLM
	Prompt     string         // pertanyaan (+ konteks chunk nanti)
	JSONSchema map[string]any // kalau diisi, jawaban WAJIB berupa JSON sesuai skema ini
}

// LLM adalah kontrak yang harus dipenuhi semua provider (Gemini, Groq, dll).
type LLM interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Generate(ctx context.Context, req GenerateRequest) (string, error)
}
