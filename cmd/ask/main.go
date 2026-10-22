package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/db"
	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
	"github.com/alinoer/ai-cs-assistant-backend/internal/rag"
)

func main() {
	question := flag.String("q", "", "pertanyaan pelanggan")
	siteKey := flag.String("site", "demo-kopi", "site_key bisnis")
	flag.Parse()
	if *question == "" {
		log.Fatal(`pakai: go run ./cmd/ask -q "tutup jam berapa?"`)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	embedder := llm.NewGeminiProvider(
		os.Getenv("GEMINI_API_KEY"),
		os.Getenv("GEMINI_EMBED_MODEL"),
		os.Getenv("GEMINI_GEN_MODEL"),
		768,
	)
	generator := llm.NewOpenAICompatibleProvider(
		os.Getenv("OPENROUTER_BASE_URL"),
		os.Getenv("OPENROUTER_API_KEY"),
		strings.Split(os.Getenv("OPENROUTER_MODELS"), ","),
	)

	svc := rag.NewService(pool, embedder, generator)
	res, err := svc.Answer(ctx, *siteKey, *question)
	if err != nil {
		log.Fatal(err)
	}

	pretty, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(pretty))
}
