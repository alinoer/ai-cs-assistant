package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Pastikan provider ini memenuhi kontrak Generator (dicek saat compile).
var _ Generator = (*OpenAICompatibleProvider)(nil)

// OpenAICompatibleProvider bisa dipakai untuk OpenRouter, Groq, atau API lain
// yang mengikuti format OpenAI Chat Completions. Bedanya hanya baseURL & apiKey.
type OpenAICompatibleProvider struct {
	baseURL    string
	apiKey     string
	models     []string // >1 model = fallback otomatis (khusus OpenRouter)
	httpClient *http.Client
}

func NewOpenAICompatibleProvider(baseURL, apiKey string, models []string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		models:     models,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ---------- Bentuk JSON (format OpenAI) ----------

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type jsonSchemaFormat struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type responseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *jsonSchemaFormat `json:"json_schema,omitempty"`
}

type chatRequest struct {
	Model          string          `json:"model,omitempty"`
	Models         []string        `json:"models,omitempty"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// apiError menyimpan status HTTP supaya kita bisa memutuskan perlu retry atau tidak.
type apiError struct {
	status int
	body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("status %d: %s", e.status, e.body)
}

// ---------- Method publik ----------

func (p *OpenAICompatibleProvider) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	body := chatRequest{Temperature: 0.2}
	if req.System != "" {
		body.Messages = append(body.Messages, chatMessage{Role: "system", Content: req.System})
	}
	body.Messages = append(body.Messages, chatMessage{Role: "user", Content: req.Prompt})

	// Satu model → field "model". Beberapa model → field "models" (fallback OpenRouter).
	if len(p.models) == 1 {
		body.Model = p.models[0]
	} else {
		body.Models = p.models
	}

	wantJSON := req.JSONSchema != nil
	if wantJSON {
		body.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchemaFormat{
				Name:   req.SchemaName,
				Strict: true,
				Schema: req.JSONSchema,
			},
		}
	}

	// Coba maksimal 2 kali. Error dari sisi kita (400/401/403) tidak di-retry.
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		text, err := p.call(ctx, body, wantJSON)
		if err == nil {
			return text, nil
		}
		lastErr = err

		var ae *apiError
		if errors.As(err, &ae) && ae.status < 500 && ae.status != 429 {
			break // salah request/key → mengulang tidak akan membantu
		}
		if ctx.Err() != nil {
			break // waktu habis
		}
	}
	return "", fmt.Errorf("generate: %w", lastErr)
}

// ---------- Satu kali panggilan HTTP + parsing defensif ----------

func (p *OpenAICompatibleProvider) call(ctx context.Context, body chatRequest, wantJSON bool) (string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", &apiError{status: resp.StatusCode, body: string(respBody)}
	}

	var out chatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("provider error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("respons tanpa choices")
	}

	choice := out.Choices[0]
	if choice.FinishReason != "stop" {
		return "", fmt.Errorf("finish_reason %q (jawaban mungkin terpotong)", choice.FinishReason)
	}

	text := choice.Message.Content
	if wantJSON {
		text = stripCodeFence(text)
		if !json.Valid([]byte(text)) {
			return "", fmt.Errorf("output bukan JSON valid: %.200s", text)
		}
	}
	return text, nil
}

// stripCodeFence membuang pembungkus ```json ... ``` kalau model menambahkannya.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[i+1:] // buang label bahasa setelah ```, mis. "json"
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
