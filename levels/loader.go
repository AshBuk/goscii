// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package levels loads mission data from the embedded archive.
package levels

import (
	"embed"
	"fmt"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AshBuk/goscii/engine"
)

//go:embed adventures
var FS embed.FS

//go:embed logs.yaml
var logsYAML []byte

// LogFact is one canonical crew log fragment.
type LogFact struct {
	ID   string `yaml:"id"`
	Fact string `yaml:"fact"`
}

// CrewLogs returns the crew log fragments in recovery order.
func CrewLogs() []LogFact {
	var out []LogFact
	_ = yaml.Unmarshal(logsYAML, &out) // embedded and covered by tests
	return out
}

type Mission struct {
	ID         string           `yaml:"id"         json:"id"`
	Title      string           `yaml:"title"      json:"title"`
	Concept    string           `yaml:"concept"    json:"concept"`
	Beat       string           `yaml:"beat"       json:"beat,omitempty"`
	Kind       Kind             `yaml:"kind"       json:"kind,omitempty"`
	Difficulty Difficulty       `yaml:"difficulty" json:"difficulty"`
	System     string           `yaml:"system"     json:"system,omitempty"` // station system the mission repairs
	Story      string           `yaml:"story"      json:"story"`
	Hints      []string         `yaml:"hints"      json:"hints"`
	Answer     string           `yaml:"answer"     json:"answer"`
	Seed       string           `yaml:"seed"       json:"seed,omitempty"`  // initial editor content (fault, refit)
	Fault      string           `yaml:"fault"      json:"fault,omitempty"` // what was planted; for the analyzer only
	Check      engine.CheckRule `yaml:"check"      json:"check"`
}

// Adventure is a named embedded campaign track.
type Adventure string

const Onboarding Adventure = "onboarding"

// BundledAdventures is the registry of adventure tracks shipped with the binary.
// A bundled track may be handcrafted or AI-generated; either way it runs offline.
var BundledAdventures = []Adventure{Onboarding}

// Missions returns the ordered level paths for this adventure.
func (a Adventure) Missions() ([]string, error) {
	root := path.Join("adventures", string(a))
	entries, err := FS.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("adventure %q not found: %w", a, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			paths = append(paths, path.Join(root, e.Name()))
		}
	}
	return paths, nil
}

// Resume returns the index of the checkpoint in paths, or 0. A checkpoint from
// before a track was renumbered still matches by the name after the NN_ prefix.
func Resume(paths []string, checkpoint string) int {
	if checkpoint == "" {
		return 0
	}
	if i := slices.Index(paths, checkpoint); i >= 0 {
		return i
	}
	slug := missionSlug(checkpoint)
	for i, p := range paths {
		if missionSlug(p) == slug {
			return i
		}
	}
	return 0
}

func missionSlug(p string) string {
	_, slug, _ := strings.Cut(path.Base(p), "_")
	return slug
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
