// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// Chat implements Provider on any OpenAI-compatible chat completions endpoint:
// OpenAI, Groq, and custom servers such as Ollama, LM Studio or OpenRouter.
type Chat struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	effort  bool // the endpoint understands reasoning_effort
	client  *http.Client
	// plainJSON flips on after the endpoint rejects json_schema; later requests
	// fall back to json_object with the schema spelled out in the system prompt.
	plainJSON atomic.Bool
}

// NewChat builds a chat-completions transport. baseURL ends before /chat/completions.
func NewChat(name, baseURL, apiKey, model string, effort bool) *Chat {
	return &Chat{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		effort:  effort,
		client:  &http.Client{Timeout: 3 * time.Minute},
	}
}

func (c *Chat) Name() string  { return c.name }
func (c *Chat) Model() string { return c.model }

type chatRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	ResponseFormat  *chatFormat   `json:"response_format,omitempty"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
	MaxTokens       int           `json:"max_completion_tokens,omitempty"`
}

type chatFormat struct {
	Type       string          `json:"type"`
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"`
}

type chatJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Refusal string `json:"refusal,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type chatError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// httpError keeps the status so callers can tell a schema rejection from an outage.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("GOSCII signal lost: HTTP %d: %s", e.status, e.msg)
}

func (c *Chat) Complete(ctx context.Context, p Prompt) (Reply, error) {
	reply, err := c.complete(ctx, p)
	var he *httpError
	if p.Schema != nil && !c.plainJSON.Load() && errors.As(err, &he) && he.status == http.StatusBadRequest {
		c.plainJSON.Store(true)
		return c.complete(ctx, p)
	}
	return reply, err
}

func (c *Chat) complete(ctx context.Context, p Prompt) (Reply, error) {
	req := chatRequest{Model: c.model, MaxTokens: p.MaxTokens}
	system := p.System
	if p.Schema != nil {
		if c.plainJSON.Load() {
			schema, _ := json.Marshal(p.Schema)
			system += "\n\nRespond with one JSON object matching this JSON schema:\n" + string(schema)
			req.ResponseFormat = &chatFormat{Type: "json_object"}
		} else {
			req.ResponseFormat = &chatFormat{
				Type:       "json_schema",
				JSONSchema: &chatJSONSchema{Name: "transmission", Schema: p.Schema, Strict: true},
			}
		}
	}
	if system != "" {
		req.Messages = append(req.Messages, chatMessage{Role: "system", Content: system})
	}
	for _, t := range p.Turns {
		req.Messages = append(req.Messages, chatMessage{Role: t.Role, Content: t.Text})
	}
	if c.effort {
		req.ReasoningEffort = p.Effort
	}

	var cr chatResponse
	if err := c.do(ctx, http.MethodPost, "/chat/completions", req, &cr); err != nil {
		return Reply{}, err
	}
	usage := Usage{Input: cr.Usage.PromptTokens, Output: cr.Usage.CompletionTokens}
	if len(cr.Choices) == 0 {
		return Reply{Usage: usage}, fmt.Errorf("GOSCII signal lost: empty response")
	}
	choice := cr.Choices[0]
	switch {
	case choice.Message.Refusal != "":
		return Reply{Usage: usage}, fmt.Errorf("%w: %s", ErrRefused, choice.Message.Refusal)
	case choice.FinishReason == "length":
		return Reply{Usage: usage}, ErrTruncated
	case choice.FinishReason == "content_filter":
		return Reply{Usage: usage}, ErrRefused
	}
	text := strings.TrimSpace(choice.Message.Content)
	if text == "" {
		return Reply{Usage: usage}, fmt.Errorf("GOSCII signal lost: empty response")
	}
	return Reply{Turn: Turn{Role: "assistant", Text: text}, Usage: usage}, nil
}

func (c *Chat) Models(ctx context.Context) ([]string, error) {
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/models", nil, &out); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// do sends one request, retrying rate limits and server faults twice.
func (c *Chat) do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		retry, err := c.send(ctx, method, path, payload, out)
		if !retry {
			return err
		}
		lastErr = err
	}
	return lastErr
}

// send performs one HTTP exchange and reports whether a failure is worth retrying.
func (c *Chat) send(ctx context.Context, method, path string, payload []byte, out any) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("GOSCII signal lost: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, fmt.Errorf("GOSCII signal lost: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return false, fmt.Errorf("%w (HTTP %d)", ErrKeyRejected, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return true, &httpError{resp.StatusCode, errorMessage(data, resp.StatusCode)}
	case resp.StatusCode >= 400:
		return false, &httpError{resp.StatusCode, errorMessage(data, resp.StatusCode)}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("GOSCII signal lost: unreadable reply: %w", err)
	}
	return false, nil
}

func errorMessage(data []byte, status int) string {
	var ce chatError
	if json.Unmarshal(data, &ce) == nil && ce.Error.Message != "" {
		return ce.Error.Message
	}
	return http.StatusText(status)
}
