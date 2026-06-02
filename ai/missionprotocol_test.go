// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"strings"
	"testing"

	"github.com/AshBuk/goscii/levels"
)

// --- validatePayload / normalizeTemplate: granular field-level checks ---

func TestValidateGeneratedAcceptsMissionPayload(t *testing.T) {
	if err := validatePayload(validGenerated()); err != nil {
		t.Fatalf("expected valid mission payload: %v", err)
	}
}

func TestValidateGeneratedRequiresCheck(t *testing.T) {
	gen := validGenerated()
	gen.Check.StdoutEquals = ""

	if err := validatePayload(gen); err == nil {
		t.Fatal("expected missing check assertion error")
	}
}

func TestValidateGeneratedRequiresFields(t *testing.T) {
	tests := []struct {
		name string
		edit func(*missionPayload)
	}{
		{"story", func(gen *missionPayload) { gen.Story = "" }},
		{"hints", func(gen *missionPayload) { gen.Hints = nil }},
		{"answer", func(gen *missionPayload) { gen.Answer = "" }},
		{"markers", func(gen *missionPayload) {
			gen.Template = "package main\n\nfunc main() {}"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validGenerated()
			tt.edit(&gen)
			if err := validatePayload(gen); err == nil {
				t.Fatalf("expected missing %s error", tt.name)
			}
		})
	}
}

func TestNormalizeTemplateAddsMissingEndMarker(t *testing.T) {
	template := `package main

func main() {
	// === YOUR CODE HERE ===
}`

	got := normalizeTemplate(template)
	if !strings.Contains(got, "// === END ===") {
		t.Fatalf("expected end marker in normalized template:\n%s", got)
	}
}

// --- parseMission: the full untrusted-JSON path (fences → decode → unescape → validate) ---

const validMissionJSON = `{
  "id": "variables-easy-starlit",
  "title": "Seal Oxygen Loop",
  "concept": "variables",
  "story": "Oxygen flickers. Name the valve. Output: green",
  "hints": ["Use a variable.", "Print it."],
  "answer": "signal := \"green\"\nfmt.Println(signal)",
  "check": {"stdout_equals": "green"},
  "template": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\t// === END ===\n}"
}`

func req() Request {
	return Request{Topic: Topic{Slug: "variables"}, Difficulty: levels.Easy}
}

func TestParseMissionValid(t *testing.T) {
	m, tmpl, err := parseMission(validMissionJSON, req())
	if err != nil {
		t.Fatalf("parseMission: %v", err)
	}
	if m.ID != "variables-easy-starlit" || m.Title != "Seal Oxygen Loop" {
		t.Fatalf("unexpected mission fields: %+v", m)
	}
	if m.Difficulty != levels.Easy {
		t.Fatalf("difficulty should come from the Request, got %q", m.Difficulty)
	}
	if m.Check.StdoutEquals != "green" {
		t.Fatalf("check not carried over: %q", m.Check.StdoutEquals)
	}
	if !strings.Contains(tmpl, "// === YOUR CODE HERE ===") {
		t.Fatalf("template lost its marker:\n%s", tmpl)
	}
}

func TestParseMissionStripsFences(t *testing.T) {
	fenced := "```json\n" + validMissionJSON + "\n```"
	if _, _, err := parseMission(fenced, req()); err != nil {
		t.Fatalf("fenced payload should parse: %v", err)
	}
}

func TestParseMissionUnescapesCheckOutput(t *testing.T) {
	// The model emits an escaped newline in the expected output; the program
	// itself prints a real newline, so the check value must be unescaped to match.
	raw := strings.Replace(validMissionJSON,
		`"check": {"stdout_equals": "green"}`,
		`"check": {"stdout_equals": "line1\\nline2"}`, 1)
	m, _, err := parseMission(raw, req())
	if err != nil {
		t.Fatalf("parseMission: %v", err)
	}
	if m.Check.StdoutEquals != "line1\nline2" {
		t.Fatalf("expected unescaped newline, got %q", m.Check.StdoutEquals)
	}
}

func TestParseMissionRejectsUndecodableInput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"not json", "GOSCII signal lost in static"},
		{"empty object", "{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := parseMission(tt.raw, req()); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func validGenerated() missionPayload {
	gen := missionPayload{
		ID:      "variables-easy-starlit",
		Title:   "Seal Oxygen Loop",
		Concept: "variables",
		Story:   "Oxygen flickers. Name the valve.",
		Hints:   []string{"Use a variable.", "Print it."},
		Answer:  `signal := "green"` + "\n" + `fmt.Println(signal)`,
		Template: `package main

import "fmt"

func main() {
	// === YOUR CODE HERE ===
	// === END ===
}`,
	}
	gen.Check.StdoutEquals = "green"
	return gen
}
