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
)

// chatRequest is the wire format for OpenAI-compatible endpoints (Groq, OpenAI).
type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	Temperature    *float64            `json:"temperature,omitempty"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
}

func tempPtr(t float64) *float64 { return &t }

type chatResponseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// callChat sends a request to an OpenAI-compatible chat completions endpoint
// and returns the first choice's text content.
func callChat(ctx context.Context, client *http.Client, endpoint, apiKey string, req chatRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("signal request: %w", err)
	}
	defer resp.Body.Close()

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("GOSCII signal lost: HTTP %d", resp.StatusCode)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("GOSCII signal lost: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("GOSCII signal lost: empty response")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}
