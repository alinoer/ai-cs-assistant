package llm

import "context"

// LLM adalah kontrak yang harus dipenuhi semua provider (Gemini, Groq, dll).
// Generate ditambahkan di Hari 2; untuk sekarang cukup Embed.
type LLM interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}
