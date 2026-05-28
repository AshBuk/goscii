// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"strings"
	"testing"
)

func TestValidateGeneratedRequiresCheck(t *testing.T) {
	gen := validGenerated()
	gen.Check.StdoutEquals = ""

	if err := validatePayload(gen); err == nil {
		t.Fatal("expected missing check assertion error")
	}
}

func TestValidateGeneratedRequiresPlayableBriefing(t *testing.T) {
	tests := []struct {
		name string
		edit func(*missionPayload)
	}{
		{
			name: "story",
			edit: func(gen *missionPayload) { gen.Story = "" },
		},
		{
			name: "hints",
			edit: func(gen *missionPayload) { gen.Hints = nil },
		},
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

func TestValidateGeneratedRequiresTemplateMarkers(t *testing.T) {
	gen := validGenerated()
	gen.Template = `package main

func main() {
}`

	if err := validatePayload(gen); err == nil {
		t.Fatal("expected missing template marker error")
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

func TestValidateGeneratedAcceptsMissionPayload(t *testing.T) {
	if err := validatePayload(validGenerated()); err != nil {
		t.Fatalf("expected valid mission payload: %v", err)
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
