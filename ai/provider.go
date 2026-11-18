// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package ai is the GOSCII signal: transports to model providers, the model
// catalog, the topic syllabus, and the prompts GOSCII speaks through.
package ai

import (
	"context"
	"errors"
)

// Turn is one message in a conversation with the signal.
type Turn struct {
	Role string // "user" or "assistant"
	Text string
	raw  any // provider-native message, replayed verbatim when the turn is sent back
}

// User builds a user turn.
func User(text string) Turn { return Turn{Role: "user", Text: text} }

// Prompt is one request to the signal.
type Prompt struct {
	System    string
	Turns     []Turn
	Schema    map[string]any // JSON schema for structured output; nil means free text
	Effort    string         // "low", "medium", "high"; empty keeps the model default
	MaxTokens int
}

// Usage counts tokens spent on one or more requests.
type Usage struct {
	Input  int
	Output int
}

// Add accumulates another usage record.
func (u *Usage) Add(o Usage) {
	u.Input += o.Input
	u.Output += o.Output
}

// Reply is the signal's answer to a Prompt.
type Reply struct {
	Turn  Turn // append to Prompt.Turns to continue the conversation
	Usage Usage
}

// Text returns the reply text.
func (r Reply) Text() string { return r.Turn.Text }

// Provider is a transport to one model on one provider.
type Provider interface {
	Name() string
	Model() string
	Complete(ctx context.Context, p Prompt) (Reply, error)
	// Models lists the model IDs the key can reach. It doubles as a key check.
	Models(ctx context.Context) ([]string, error)
}

var (
	// ErrKeyRejected means the provider refused the credentials.
	ErrKeyRejected = errors.New("signal rejected the key")
	// ErrTruncated means the reply hit the token ceiling before it finished.
	ErrTruncated = errors.New("transmission cut off at the token ceiling")
	// ErrRefused means the model declined the request.
	ErrRefused = errors.New("signal refused the transmission")
)
