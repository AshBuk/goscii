// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package ai provides AI-driven level generation for GOSCII practice missions.
package ai

import (
	"context"

	"github.com/AshBuk/goscii/levels"
)

// Topic is a Go concept the player can practice.
type Topic struct {
	Slug     string
	Title    string
	Concepts string // short comma-separated display string
}

// Selection is what the entry screen returns to the caller.
type Selection struct {
	Topic      Topic
	Difficulty levels.Difficulty
}

// Request is sent to a Provider to generate one mission.
type Request struct {
	Topic      Topic
	Difficulty levels.Difficulty
	Extra      string // optional extra instructions (future: custom system prompt)
}

// AnalyzeRequest is sent to a Provider to analyze a compiler or runtime error.
type AnalyzeRequest struct {
	Concept string // Go topic being practiced
	Code    string // player's code
	ErrMsg  string // compiler stderr or wrong-output message
}

// Provider generates Go coding missions and analyzes errors on demand.
type Provider interface {
	// Generate returns a Mission and its Go template for the cockpit.
	Generate(ctx context.Context, req Request) (*levels.Mission, string, error)
	// Analyze returns a GOSCII-voiced explanation of a compiler/runtime error.
	Analyze(ctx context.Context, req AnalyzeRequest) (string, error)
	Name() string
	Models() []string
}
