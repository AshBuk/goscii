// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AshBuk/goscii/levels"
)

const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
)

// AnthropicModels is the ordered list of models shown in the config UI.
var AnthropicModels = []string{
	"claude-opus-4-8",
	"claude-sonnet-4-6",
	"claude-haiku-4-5-20251001",
}

const anthropicDefaultModel = "claude-sonnet-4-6"

// Anthropic implements Provider using the Anthropic Messages API.
type Anthropic struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropic(apiKey, model string) *Anthropic {
	if model == "" {
		model = anthropicDefaultModel
	}
	return &Anthropic{apiKey: apiKey, model: model, client: &http.Client{Timeout: 60 * time.Second}}
}

func (a *Anthropic) Name() string     { return "anthropic" }
func (a *Anthropic) Models() []string { return AnthropicModels }

// --- API types ---

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *Anthropic) call(ctx context.Context, req anthropicRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("signal request: %w", err)
	}
	defer resp.Body.Close()

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return "", fmt.Errorf("GOSCII signal lost: HTTP %d", resp.StatusCode)
	}
	if ar.Error != nil {
		return "", fmt.Errorf("GOSCII signal lost: %s", ar.Error.Message)
	}
	for _, block := range ar.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("GOSCII signal lost: empty response")
}

// Analyze calls the Anthropic API and returns a GOSCII-voiced error explanation.
func (a *Anthropic) Analyze(ctx context.Context, req AnalyzeRequest) (string, error) {
	return a.call(ctx, anthropicRequest{
		Model:     a.model,
		MaxTokens: 512,
		System:    analyzeSystemPrompt(),
		Messages:  []anthropicMessage{{Role: "user", Content: analyzeUserMessage(req)}},
	})
}

// Generate calls the Anthropic API and returns a Mission + Go template ready for the cockpit.
func (a *Anthropic) Generate(ctx context.Context, req Request) (*levels.Mission, string, error) {
	raw, err := a.call(ctx, anthropicRequest{
		Model:     a.model,
		MaxTokens: 2048,
		System:    systemPrompt(),
		Messages:  []anthropicMessage{{Role: "user", Content: userMessage(req)}},
	})
	if err != nil {
		return nil, "", err
	}
	return parseMission(raw, req)
}
