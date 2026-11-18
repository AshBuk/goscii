// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// Anthropic implements Provider on the Claude Messages API.
type Anthropic struct {
	client anthropic.Client
	model  string
}

func NewAnthropic(apiKey, model string, opts ...option.RequestOption) *Anthropic {
	opts = append([]option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(3 * time.Minute),
		option.WithMaxRetries(2),
	}, opts...)
	return &Anthropic{client: anthropic.NewClient(opts...), model: model}
}

func (a *Anthropic) Name() string  { return "anthropic" }
func (a *Anthropic) Model() string { return a.model }

// fallbackReady lists model families that accept server-side refusal fallbacks.
var fallbackReady = []string{"claude-fable-5", "claude-mythos-5", "claude-opus-5", "claude-sonnet-5-5"}

func (a *Anthropic) Complete(ctx context.Context, p Prompt) (Reply, error) {
	maxTokens := p.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16000 // thinking shares this ceiling on current Claude models
	}
	params := anthropic.BetaMessageNewParams{
		Model:     a.model,
		MaxTokens: int64(maxTokens),
		Messages:  make([]anthropic.BetaMessageParam, 0, len(p.Turns)),
	}
	if p.System != "" {
		params.System = []anthropic.BetaTextBlockParam{{Text: p.System}}
	}
	for _, t := range p.Turns {
		params.Messages = append(params.Messages, anthropicTurn(t))
	}
	if p.Schema != nil {
		params.OutputConfig.Format = anthropic.BetaJSONOutputFormatParam{Schema: p.Schema}
	}
	if p.Effort != "" {
		params.OutputConfig.Effort = anthropic.BetaOutputConfigEffort(p.Effort)
	}
	for _, prefix := range fallbackReady {
		if strings.HasPrefix(a.model, prefix) {
			params.Betas = append(params.Betas, anthropic.AnthropicBetaServerSideFallback2026_07_01)
			params.Fallbacks = anthropic.BetaFallbacksParamOfDefault()
			break
		}
	}

	msg, err := a.client.Beta.Messages.New(ctx, params)
	if err != nil {
		return Reply{}, anthropicErr(err)
	}
	usage := Usage{Input: int(msg.Usage.InputTokens), Output: int(msg.Usage.OutputTokens)}
	switch msg.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return Reply{Usage: usage}, fmt.Errorf("%w: %s", ErrRefused, msg.StopDetails.Explanation)
	case anthropic.BetaStopReasonMaxTokens:
		return Reply{Usage: usage}, ErrTruncated
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return Reply{Usage: usage}, fmt.Errorf("GOSCII signal lost: empty response")
	}
	turn := Turn{Role: "assistant", Text: strings.TrimSpace(text.String()), raw: msg.ToParam()}
	return Reply{Turn: turn, Usage: usage}, nil
}

// anthropicTurn replays an assistant turn verbatim (thinking blocks included)
// when it came from this provider; otherwise it rebuilds the message from text.
func anthropicTurn(t Turn) anthropic.BetaMessageParam {
	if raw, ok := t.raw.(anthropic.BetaMessageParam); ok {
		return raw
	}
	block := anthropic.NewBetaTextBlock(t.Text)
	if t.Role == "assistant" {
		return anthropic.BetaMessageParam{OfAssistant: &anthropic.BetaAssistantMessageParam{
			Content: []anthropic.BetaContentBlockParamUnion{block},
		}}
	}
	return anthropic.NewBetaUserMessage(block)
}

func (a *Anthropic) Models(ctx context.Context) ([]string, error) {
	pager := a.client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	var ids []string
	for pager.Next() {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		return nil, anthropicErr(err)
	}
	return ids, nil
}

func anthropicErr(err error) error {
	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("GOSCII signal lost: %w", err)
	}
	if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w (HTTP %d)", ErrKeyRejected, apiErr.StatusCode)
	}
	msg := apiErr.Message
	if msg == "" {
		msg = http.StatusText(apiErr.StatusCode)
	}
	return fmt.Errorf("GOSCII signal lost: HTTP %d: %s", apiErr.StatusCode, msg)
}
