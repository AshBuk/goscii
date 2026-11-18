// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"fmt"
	"slices"

	"github.com/AshBuk/goscii/engine"
)

// Band is one provider's frequency: where to reach it and which models GOSCII
// has verified against the mission bench. Any other model ID the provider
// serves can still be dialed by hand.
type Band struct {
	Provider engine.Provider
	BaseURL  string   // chat-completions base; empty for Anthropic and custom
	Verified []string // first entry is the default
	// Retired maps models dropped from the catalog to their successor, so an
	// old config keeps working after an upgrade.
	Retired map[string]string
	effort  bool // accepts reasoning_effort on chat completions
}

// Default returns the band's recommended model, or "" when there is none.
func (b Band) Default() string {
	if len(b.Verified) == 0 {
		return ""
	}
	return b.Verified[0]
}

// Resolve maps an empty or retired model ID to the one GOSCII should dial.
func (b Band) Resolve(model string) string {
	if model == "" {
		return b.Default()
	}
	if next, ok := b.Retired[model]; ok {
		return next
	}
	return model
}

// Bands is the model catalog, checked against provider docs on 2026-11-17.
var Bands = map[engine.Provider]Band{
	engine.ProviderAnthropic: {
		Provider: engine.ProviderAnthropic,
		Verified: []string{"claude-sonnet-5-5", "claude-opus-5-5", "claude-haiku-5-5"},
		Retired: map[string]string{
			"claude-opus-4-8":           "claude-opus-5-5",
			"claude-sonnet-4-6":         "claude-sonnet-5-5",
			"claude-haiku-4-5-20251001": "claude-haiku-5-5",
		},
	},
	engine.ProviderOpenAI: {
		Provider: engine.ProviderOpenAI,
		BaseURL:  "https://api.openai.com/v1",
		Verified: []string{"gpt-6.1-sol", "gpt-6.1-terra", "gpt-6-luna", "gpt-6-astra"},
		Retired: map[string]string{
			"gpt-5":        "gpt-6.1-sol",
			"gpt-5-mini":   "gpt-6.1-terra",
			"gpt-4.1":      "gpt-6-luna",
			"gpt-4.1-mini": "gpt-6-luna",
			"gpt-4o":       "gpt-6-luna",
			"gpt-4o-mini":  "gpt-6-luna",
			"o4-mini":      "gpt-6.1-terra",
			"o3-mini":      "gpt-6.1-sol",
		},
		effort: true,
	},
	engine.ProviderGroq: {
		Provider: engine.ProviderGroq,
		BaseURL:  "https://api.groq.com/openai/v1",
		Verified: []string{"openai/gpt-oss-120b", "openai/gpt-oss-20b"},
		Retired: map[string]string{
			"llama-3.3-70b-versatile": "openai/gpt-oss-120b",
			"llama-3.1-8b-instant":    "openai/gpt-oss-20b",
		},
		effort: true,
	},
	engine.ProviderCustom: {Provider: engine.ProviderCustom},
}

// IsVerified reports whether model is on the band's verified list.
func (b Band) IsVerified(model string) bool { return slices.Contains(b.Verified, model) }

// Dial opens a signal from the stored config.
func Dial(cfg engine.Config) (Provider, error) {
	band, ok := Bands[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("signal %q is not wired yet", cfg.Provider)
	}
	model := band.Resolve(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("signal %q has no model selected", cfg.Provider)
	}
	switch cfg.Provider {
	case engine.ProviderAnthropic:
		return NewAnthropic(cfg.Key(), model), nil
	case engine.ProviderCustom:
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("custom signal has no endpoint")
		}
		return NewChat(string(cfg.Provider), cfg.BaseURL, cfg.Key(), model, false), nil
	default:
		return NewChat(string(cfg.Provider), band.BaseURL, cfg.Key(), model, band.effort), nil
	}
}
