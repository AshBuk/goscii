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

const openaiEndpoint = "https://api.openai.com/v1/chat/completions"

// OpenAIModels is the ordered list of models shown in the config UI.
var OpenAIModels = []string{
	"gpt-5",
	"gpt-5-mini",
	"gpt-4.1",
	"gpt-4.1-mini",
	"gpt-4o",
	"gpt-4o-mini",
	"o4-mini",
	"o3-mini",
}

// OpenAI implements Provider using the OpenAI chat completions API.
type OpenAI struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAI(apiKey, model string) *OpenAI {
	if model == "" {
		model = OpenAIModels[0]
	}
	return &OpenAI{apiKey: apiKey, model: model, client: &http.Client{Timeout: 60 * time.Second}}
}

func (o *OpenAI) Name() string     { return "openai" }
func (o *OpenAI) Models() []string { return OpenAIModels }

// isReasoningModel reports whether model is an OpenAI o-series reasoning model
// that does not accept the temperature parameter.
func isReasoningModel(model string) bool {
	return len(model) > 1 && model[0] == 'o' && model[1] >= '1' && model[1] <= '9'
}

// Analyze calls the OpenAI API and returns a GOSCII-voiced error explanation.
func (o *OpenAI) Analyze(ctx context.Context, req AnalyzeRequest) (string, error) {
	return callChat(ctx, o.client, openaiEndpoint, o.apiKey, chatRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "system", Content: analyzeSystemPrompt()},
			{Role: "user", Content: analyzeUserMessage(req)},
		},
		Temperature: tempOrNil(o.model, 0.7),
	})
}

// Generate calls the OpenAI API and returns a Mission + Go template ready for the cockpit.
func (o *OpenAI) Generate(ctx context.Context, req Request) (*levels.Mission, string, error) {
	raw, err := callChat(ctx, o.client, openaiEndpoint, o.apiKey, chatRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: userMessage(req)},
		},
		Temperature:    tempOrNil(o.model, 0.9),
		ResponseFormat: &chatResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return nil, "", err
	}
	return parseMission(raw, req)
}

// tempOrNil returns a temperature pointer for non-reasoning models, nil for o-series.
func tempOrNil(model string, t float64) *float64 {
	if isReasoningModel(model) {
		return nil
	}
	return tempPtr(t)
}
