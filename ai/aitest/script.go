// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package aitest provides a scripted signal for tests: no keys, no network.
package aitest

import (
	"context"
	"errors"
	"sync"

	"github.com/AshBuk/goscii/ai"
)

// Step is one scripted answer: text, or an error.
type Step struct {
	Text string
	Err  error
}

// Script replays its steps in order and records every prompt it receives.
type Script struct {
	mu      sync.Mutex
	steps   []Step
	Prompts []ai.Prompt
	ModelID string
}

// New returns a script that answers with the given texts in order.
func New(texts ...string) *Script {
	s := &Script{ModelID: "scripted"}
	for _, t := range texts {
		s.steps = append(s.steps, Step{Text: t})
	}
	return s
}

// Then appends a step.
func (s *Script) Then(st Step) *Script {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append(s.steps, st)
	return s
}

func (s *Script) Name() string  { return "script" }
func (s *Script) Model() string { return s.ModelID }

func (s *Script) Complete(_ context.Context, p ai.Prompt) (ai.Reply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Prompts = append(s.Prompts, p)
	if len(s.steps) == 0 {
		return ai.Reply{}, errors.New("script exhausted")
	}
	st := s.steps[0]
	s.steps = s.steps[1:]
	if st.Err != nil {
		return ai.Reply{}, st.Err
	}
	return ai.Reply{Turn: ai.Turn{Role: "assistant", Text: st.Text}, Usage: ai.Usage{Input: 100, Output: 50}}, nil
}

func (s *Script) Models(context.Context) ([]string, error) { return []string{s.ModelID}, nil }

// Calls returns how many prompts the script has answered.
func (s *Script) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Prompts)
}
