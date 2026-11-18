// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Provider string

const (
	ProviderGroq      Provider = "groq"
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderCustom    Provider = "custom" // any OpenAI-compatible endpoint: Ollama, LM Studio, OpenRouter
)

// Providers is the order shown in the signal setup screen.
var Providers = []Provider{ProviderAnthropic, ProviderOpenAI, ProviderGroq, ProviderCustom}

// keyEnv lists the environment variables checked, in order, when no key is stored.
var keyEnv = map[Provider][]string{
	ProviderAnthropic: {"GOSCII_API_KEY", "ANTHROPIC_API_KEY"},
	ProviderOpenAI:    {"GOSCII_API_KEY", "OPENAI_API_KEY"},
	ProviderGroq:      {"GOSCII_API_KEY", "GROQ_API_KEY"},
	ProviderCustom:    {"GOSCII_API_KEY"},
}

type Config struct {
	Provider Provider `json:"provider"`
	APIKey   string   `json:"api_key,omitempty"`
	Model    string   `json:"model"`
	BaseURL  string   `json:"base_url,omitempty"`
	// Prefetch forges the next chain mission while the current one is in play.
	// Nil means on.
	Prefetch *bool `json:"prefetch,omitempty"`
}

// Key returns the stored API key, falling back to the provider's environment variables.
func (c Config) Key() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return EnvKey(c.Provider)
}

// KeySource names where Key comes from: "config", an env variable name, or "".
func (c Config) KeySource() string {
	if c.APIKey != "" {
		return "config"
	}
	for _, name := range keyEnv[c.Provider] {
		if os.Getenv(name) != "" {
			return name
		}
	}
	return ""
}

// Wired reports whether the config can reach a signal. A custom endpoint may run
// without a key (local Ollama); every hosted provider needs one.
func (c Config) Wired() bool {
	if c.Provider == ProviderCustom {
		return c.BaseURL != ""
	}
	return c.Key() != ""
}

// PrefetchOn reports whether background forging is enabled.
func (c Config) PrefetchOn() bool { return c.Prefetch == nil || *c.Prefetch }

// EnvKey returns the first non-empty key variable for the provider.
func EnvKey(p Provider) string {
	for _, name := range keyEnv[p] {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

func configPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads config.json. Without one, it picks the first provider whose
// key is present in the environment, so `docker run -e ANTHROPIC_API_KEY=...`
// works with no setup.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return envConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

func envConfig() *Config {
	for _, p := range []Provider{ProviderAnthropic, ProviderOpenAI, ProviderGroq} {
		for _, name := range keyEnv[p][1:] {
			if os.Getenv(name) != "" {
				return &Config{Provider: p}
			}
		}
	}
	return &Config{Provider: ProviderAnthropic}
}

func SaveConfig(c *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
