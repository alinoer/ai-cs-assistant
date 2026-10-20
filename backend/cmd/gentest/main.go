package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
)

func main() {
	var gen llm.Generator = llm.NewOpenAICompatibleProvider(
		os.Getenv("OPENROUTER_BASE_URL"),
		os.Getenv("OPENROUTER_API_KEY"),
		strings.Split(os.Getenv("OPENROUTER_MODELS"), ","),
	)

	// JSON Schema standar (huruf kecil), format OpenAI
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"answer":   map[string]any{"type": "string"},
			"answered": map[string]any{"type": "boolean"},
		},
		"required":             []string{"answer", "answered"},
		"additionalProperties": false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	raw, err := gen.Generate(ctx, llm.GenerateRequest{
		System:     "Kamu asisten customer service. Jawab singkat dalam bahasa Indonesia.",
		Prompt:     "Apa itu kopi susu gula aren? Jawab satu kalimat.",
		SchemaName: "cs_answer",
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
