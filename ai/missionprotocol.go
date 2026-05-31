// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// missionPayload mirrors the JSON schema the AI is prompted to produce.
type missionPayload struct {
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

// parseMission decodes a raw JSON string into a Mission and its Go template.
func parseMission(raw string, req Request) (*levels.Mission, string, error) {
	raw = stripFences(raw)
	var gen missionPayload
	if err := json.Unmarshal([]byte(raw), &gen); err != nil {
		return nil, "", fmt.Errorf("parse generated level: %w\nraw: %.500s", err, raw)
	}
	gen.Template = normalizeTemplate(gen.Template)
	gen.Check.StdoutEquals = unescapeOutput(gen.Check.StdoutEquals)
	gen.Check.StdoutContains = unescapeOutput(gen.Check.StdoutContains)
	if err := validatePayload(gen); err != nil {
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

func validatePayload(gen missionPayload) error {
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

func unescapeOutput(s string) string {
	r := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\r`, "\r")
	return r.Replace(s)
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
