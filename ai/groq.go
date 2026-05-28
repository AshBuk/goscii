// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"context"
	"net/http"
	"time"

	"github.com/AshBuk/goscii/levels"
)

const groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// GroqModels is the ordered list of models shown in the config UI.
var GroqModels = []string{
	"llama-3.3-70b-versatile",
}

// Groq implements Provider using the Groq OpenAI-compatible API.
type Groq struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGroq(apiKey, model string) *Groq {
	if model == "" {
		model = GroqModels[0]
	}
	return &Groq{apiKey: apiKey, model: model, client: &http.Client{Timeout: 60 * time.Second}}
}

func (g *Groq) Name() string     { return "groq" }
func (g *Groq) Models() []string { return GroqModels }

// Analyze calls the Groq API and returns a GOSCII-voiced error explanation.
func (g *Groq) Analyze(ctx context.Context, req AnalyzeRequest) (string, error) {
	return callChat(ctx, g.client, groqEndpoint, g.apiKey, chatRequest{
		Model: g.model,
		Messages: []chatMessage{
			{Role: "system", Content: analyzeSystemPrompt()},
			{Role: "user", Content: analyzeUserMessage(req)},
		},
		Temperature: tempPtr(0.7),
	})
}

// Generate calls the Groq API and returns a Mission + Go template ready for the cockpit.
func (g *Groq) Generate(ctx context.Context, req Request) (*levels.Mission, string, error) {
	raw, err := callChat(ctx, g.client, groqEndpoint, g.apiKey, chatRequest{
		Model: g.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: userMessage(req)},
		},
		Temperature:    tempPtr(0.9),
		ResponseFormat: &chatResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return nil, "", err
	}
	return parseMission(raw, req)
}
