package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/alinoer/ai-cs-assistant-backend/internal/db"
	"github.com/alinoer/ai-cs-assistant-backend/internal/llm"
	"github.com/alinoer/ai-cs-assistant-backend/internal/rag"
)

const maxQuestionChars = 500

type chatRequest struct {
	SiteKey  string `json:"site_key"`
	Question string `json:"question"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	// ctx ini otomatis "dibatalkan" saat server menerima Ctrl+C / sinyal stop
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /chat", chatHandler(svc))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      70 * time.Second, // lebih lama dari timeout jawaban (60 detik)
	}

	// Jalankan server di goroutine supaya main bisa menunggu sinyal stop
	go func() {
		log.Printf("server jalan di http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done() // tunggu Ctrl+C
	log.Println("mematikan server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func chatHandler(svc *rag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10) // body maksimal 8 KB

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{`body harus JSON: {"site_key": "...", "question": "..."}`})
			return
		}
		if strings.TrimSpace(req.SiteKey) == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{"site_key wajib diisi"})
			return
		}
		if utf8.RuneCountInString(req.Question) > maxQuestionChars {
			writeJSON(w, http.StatusBadRequest, errorResponse{"pertanyaan terlalu panjang (maks 500 karakter)"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		start := time.Now()
		res, err := svc.Answer(ctx, req.SiteKey, req.Question)
		if errors.Is(err, rag.ErrEmptyQuestion) {
			writeJSON(w, http.StatusBadRequest, errorResponse{"pertanyaan wajib diisi"})
			return
		}
		if err != nil {
			// Detail error hanya di log server, bukan dikirim ke pelanggan
			log.Printf("chat error site=%s: %v", req.SiteKey, err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{"maaf, terjadi kesalahan. silakan coba lagi sebentar"})
			return
		}

		// Isi pertanyaan sengaja TIDAK di-log (privasi pelanggan)
		log.Printf("chat site=%s answered=%v sources=%d dur=%s",
			req.SiteKey, res.Answered, len(res.Sources), time.Since(start).Round(time.Millisecond))
		writeJSON(w, http.StatusOK, res)
	}
}

// withCORS mengizinkan widget di domain lain memanggil API ini.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // jawab preflight browser
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
