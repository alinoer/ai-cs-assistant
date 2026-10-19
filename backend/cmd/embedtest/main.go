package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_EMBED_MODEL")
	if apiKey == "" || model == "" {
		log.Fatal("GEMINI_API_KEY dan GEMINI_EMBED_MODEL wajib diisi (jalankan: set -a; source .env; set +a)")
	}

	var provider llm.LLM = llm.NewGeminiProvider(apiKey, model, os.Getenv("GEMINI_GEN_MODEL"), 768)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vec, err := provider.Embed(ctx, "Kami buka setiap hari pukul 08.00 sampai 21.00")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("dimensi: %d\n5 angka pertama: %v\n", len(vec), vec[:5])
}
