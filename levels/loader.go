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

type Mission struct {
	ID      string            `yaml:"id"`
	Title   string            `yaml:"title"`
	Concept string            `yaml:"concept"`
	Story   string            `yaml:"story"`
	Hints   []string        `yaml:"hints"`
	Answer  string          `yaml:"answer"`
	Check   engine.CheckRule `yaml:"check"`
}

// Adventure is a named embedded campaign track.
type Adventure string

// Missions returns the ordered level paths for this adventure.
func (a Adventure) Missions() ([]string, error) {
	entries, err := FS.ReadDir(string(a))
	if err != nil {
		return nil, fmt.Errorf("adventure %q not found: %w", a, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			paths = append(paths, path.Join(string(a), e.Name()))
		}
	}
	return paths, nil
}

func Load(levelPath string) (*Mission, string, error) {
	yamlData, err := FS.ReadFile(path.Join(levelPath, "level.yaml"))
	if err != nil {
		return nil, "", fmt.Errorf("read level.yaml: %w", err)
	}
	var m Mission
	if err := yaml.Unmarshal(yamlData, &m); err != nil {
		return nil, "", fmt.Errorf("parse level.yaml: %w", err)
	}

	tmplData, err := FS.ReadFile(path.Join(levelPath, "template.txt"))
	if err != nil {
		return nil, "", fmt.Errorf("read template.txt: %w", err)
	}

	return &m, string(tmplData), nil
}
