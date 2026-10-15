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

// Pastikan GeminiProvider memenuhi interface LLM (dicek saat compile).
var _ LLM = (*GeminiProvider)(nil)

type GeminiProvider struct {
	apiKey     string
	embedModel string
	embedDim   int
	httpClient *http.Client
}

func NewGeminiProvider(apiKey, embedModel string, embedDim int) *GeminiProvider {
	return &GeminiProvider{
		apiKey:     apiKey,
		embedModel: embedModel,
		embedDim:   embedDim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Bentuk JSON request & response, sama seperti body curl kemarin.
type part struct {
	Text string `json:"text"`
}

type content struct {
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

func (g *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{
		Content:              content{Parts: []part{{Text: text}}},
		OutputDimensionality: g.embedDim,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:embedContent", geminiBaseURL, g.embedModel)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini embed: status %d: %s", resp.StatusCode, respBody)
	}

	var out embedResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Embedding.Values) != g.embedDim {
		return nil, fmt.Errorf("gemini embed: expected %d dims, got %d", g.embedDim, len(out.Embedding.Values))
	}
	return out.Embedding.Values, nil
}
