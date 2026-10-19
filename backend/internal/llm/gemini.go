package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

var _ LLM = (*GeminiProvider)(nil)

type GeminiProvider struct {
	apiKey     string
	embedModel string
	genModel   string
	embedDim   int
	httpClient *http.Client
}

func NewGeminiProvider(apiKey, embedModel, genModel string, embedDim int) *GeminiProvider {
	return &GeminiProvider{
		apiKey:     apiKey,
		embedModel: embedModel,
		genModel:   genModel,
		embedDim:   embedDim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ---------- Bentuk JSON ----------

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type embedRequest struct {
	Content              content `json:"content"`
	OutputDimensionality int     `json:"outputDimensionality"`
}

type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

type generationConfig struct {
	Temperature      float64        `json:"temperature"`
	ResponseMimeType string         `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]any `json:"responseSchema,omitempty"`
}

type generateRequest struct {
	SystemInstruction *content         `json:"systemInstruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type generateResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

// ---------- Method publik ----------

func (g *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	var out embedResponse
	err := g.post(ctx, g.embedModel+":embedContent", embedRequest{
		Content:              content{Parts: []part{{Text: text}}},
		OutputDimensionality: g.embedDim,
	}, &out)
	if err != nil {
		return nil, fmt.Errorf("gemini embed: %w", err)
	}
	if len(out.Embedding.Values) != g.embedDim {
		return nil, fmt.Errorf("gemini embed: expected %d dims, got %d", g.embedDim, len(out.Embedding.Values))
	}
	return out.Embedding.Values, nil
}

func (g *GeminiProvider) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	body := generateRequest{
		Contents:         []content{{Role: "user", Parts: []part{{Text: req.Prompt}}}},
		GenerationConfig: generationConfig{Temperature: 0.2}, // rendah = jawaban konsisten, tidak "kreatif"
	}
	if req.System != "" {
		body.SystemInstruction = &content{Parts: []part{{Text: req.System}}}
	}
	if req.JSONSchema != nil {
		body.GenerationConfig.ResponseMimeType = "application/json"
		body.GenerationConfig.ResponseSchema = req.JSONSchema
	}

	var out generateResponse
	if err := g.post(ctx, g.genModel+":generateContent", body, &out); err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini generate: respons kosong")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

// ---------- Helper HTTP (dipakai Embed & Generate) ----------

func (g *GeminiProvider) post(ctx context.Context, method string, reqBody, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s", geminiBaseURL, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, respBody)
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
