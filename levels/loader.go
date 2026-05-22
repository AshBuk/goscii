// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package levels loads mission data from the embedded archive.
package levels

import (
	"embed"
	"fmt"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/AshBuk/goscii/engine"
)

//go:embed onboarding
var FS embed.FS

type Level struct {
	ID      string            `yaml:"id"`
	Title   string            `yaml:"title"`
	Concept string            `yaml:"concept"`
	Story   string            `yaml:"story"`
	Hints   []string          `yaml:"hints"`
	Answer  string            `yaml:"answer"`
	Check   engine.CheckRule   `yaml:"check"`
	Capture map[string]string `yaml:"capture"`
}

func Load(levelPath string) (*Level, string, error) {
	yamlData, err := FS.ReadFile(path.Join(levelPath, "level.yaml"))
	if err != nil {
		return nil, "", fmt.Errorf("read level.yaml: %w", err)
	}
	var l Level
	if err := yaml.Unmarshal(yamlData, &l); err != nil {
		return nil, "", fmt.Errorf("parse level.yaml: %w", err)
	}

	tmplData, err := FS.ReadFile(path.Join(levelPath, "template.txt"))
	if err != nil {
		return nil, "", fmt.Errorf("read template.txt: %w", err)
	}

	return &l, string(tmplData), nil
}
