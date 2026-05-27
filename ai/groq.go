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

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

const groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// GroqModels is the ordered list of models shown in the config UI.
var GroqModels = []string{
	"llama-3.3-70b-versatile",
	"llama-3.1-8b-instant",
	"gemma2-9b-it",
	"mixtral-8x7b-32768",
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
	return &Groq{apiKey: apiKey, model: model, client: &http.Client{}}
}

func (g *Groq) Name() string     { return "groq" }
func (g *Groq) Models() []string { return GroqModels }

// --- API types ---

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// generatedJSON mirrors the JSON schema the AI is prompted to produce.
type generatedJSON struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Concept string   `json:"concept"`
	Story   string   `json:"story"`
	Hints   []string `json:"hints"`
	Answer  string   `json:"answer"`
	Check   struct {
		StdoutNonempty bool   `json:"stdout_nonempty,omitempty"`
		StdoutContains string `json:"stdout_contains,omitempty"`
		StdoutEquals   string `json:"stdout_equals,omitempty"`
	} `json:"check"`
	Template string `json:"template"`
}

// Analyze calls the Groq API and returns a GOSCII-voiced error explanation.
func (g *Groq) Analyze(ctx context.Context, req AnalyzeRequest) (string, error) {
	body, err := json.Marshal(groqRequest{
		Model: g.model,
		Messages: []groqMessage{
			{Role: "system", Content: analyzeSystemPrompt()},
			{Role: "user", Content: analyzeUserMessage(req)},
		},
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("decode groq response: %w", err)
	}
	if gr.Error != nil {
		return "", fmt.Errorf("groq: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("groq: empty response")
	}
	return strings.TrimSpace(gr.Choices[0].Message.Content), nil
}

// Generate calls the Groq API and returns a Mission + Go template ready for the cockpit.
func (g *Groq) Generate(ctx context.Context, req Request) (*levels.Mission, string, error) {
	body, err := json.Marshal(groqRequest{
		Model: g.model,
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: userMessage(req)},
		},
		Temperature: 0.9,
	})
	if err != nil {
		return nil, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, "", fmt.Errorf("decode groq response: %w", err)
	}
	if gr.Error != nil {
		return nil, "", fmt.Errorf("groq: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return nil, "", fmt.Errorf("groq: empty response")
	}

	raw := stripFences(gr.Choices[0].Message.Content)
	var gen generatedJSON
	if err := json.Unmarshal([]byte(raw), &gen); err != nil {
		return nil, "", fmt.Errorf("parse generated level: %w\nraw: %.500s", err, raw)
	}
	gen.Template = normalizeTemplate(gen.Template)
	if err := validateGenerated(gen); err != nil {
		return nil, "", fmt.Errorf("validate generated level: %w", err)
	}

	m := &levels.Mission{
		ID:         gen.ID,
		Title:      gen.Title,
		Concept:    gen.Concept,
		Difficulty: req.Difficulty,
		Story:      gen.Story,
		Hints:      gen.Hints,
		Answer:     gen.Answer,
		Check: engine.CheckRule{
			StdoutNonempty: gen.Check.StdoutNonempty,
			StdoutContains: gen.Check.StdoutContains,
			StdoutEquals:   gen.Check.StdoutEquals,
		},
	}
	return m, gen.Template, nil
}

func validateGenerated(gen generatedJSON) error {
	switch {
	case strings.TrimSpace(gen.ID) == "":
		return fmt.Errorf("missing id")
	case strings.TrimSpace(gen.Title) == "":
		return fmt.Errorf("missing title")
	case strings.TrimSpace(gen.Concept) == "":
		return fmt.Errorf("missing concept")
	case strings.TrimSpace(gen.Story) == "":
		return fmt.Errorf("missing story")
	case len(gen.Hints) == 0:
		return fmt.Errorf("missing hints")
	case strings.TrimSpace(gen.Template) == "":
		return fmt.Errorf("missing template")
	case !strings.Contains(gen.Template, "// === YOUR CODE HERE ==="):
		return fmt.Errorf("template missing player code marker")
	case !strings.Contains(gen.Template, "// === END ==="):
		return fmt.Errorf("template missing end marker")
	case strings.TrimSpace(gen.Answer) == "":
		return fmt.Errorf("missing answer")
	case !gen.Check.StdoutNonempty && gen.Check.StdoutContains == "" && gen.Check.StdoutEquals == "":
		return fmt.Errorf("missing check assertion")
	default:
		return nil
	}
}

func normalizeTemplate(template string) string {
	if !strings.Contains(template, "// === YOUR CODE HERE ===") || strings.Contains(template, "// === END ===") {
		return template
	}
	return strings.Replace(template, "// === YOUR CODE HERE ===", "// === YOUR CODE HERE ===\n\t// === END ===", 1)
}

// stripFences removes markdown code fences that some models add despite instructions.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.Index(s, "\n"); i > 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "```"); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
