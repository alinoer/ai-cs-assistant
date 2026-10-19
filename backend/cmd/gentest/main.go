package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
)

func main() {
	provider := llm.NewGeminiProvider(
		os.Getenv("GEMINI_API_KEY"),
		os.Getenv("GEMINI_EMBED_MODEL"),
		os.Getenv("GEMINI_GEN_MODEL"),
		768,
	)

	schema := map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"answer":   map[string]any{"type": "STRING"},
			"answered": map[string]any{"type": "BOOLEAN"},
		},
		"required": []string{"answer", "answered"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	raw, err := provider.Generate(ctx, llm.GenerateRequest{
		System:     "Kamu adalah asisten customer service. Jawab singkat dalam bahasa Indonesia.",
		Prompt:     "Apa itu kopi susu gula aren? Jawab dalam satu kalimat.",
		JSONSchema: schema,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("raw JSON:", raw)

	var res struct {
		Answer   string `json:"answer"`
		Answered bool   `json:"answered"`
	}
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		log.Fatalf("bukan JSON valid: %v", err)
	}
	fmt.Printf("answer: %s\nanswered: %v\n", res.Answer, res.Answered)
}

