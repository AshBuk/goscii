// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package levels

import (
	"strings"
	"testing"
)

func TestHandcraftedMissionsAreLoadable(t *testing.T) {
	for _, a := range HandcraftedAdventures {
		t.Run(string(a), func(t *testing.T) {
			paths, err := a.Missions()
			if err != nil {
				t.Fatalf("load adventure: %v", err)
			}
			if len(paths) == 0 {
				t.Fatal("expected at least one mission")
			}
			for _, p := range paths {
				t.Run(p, func(t *testing.T) {
					m, tmpl, err := Load(p)
					if err != nil {
						t.Fatalf("load mission: %v", err)
					}
					assertMission(t, m, tmpl)
				})
			}
		})
	}
}

func assertMission(t *testing.T, m *Mission, tmpl string) {
	t.Helper()
	if strings.TrimSpace(m.ID) == "" {
		t.Fatal("mission id is empty")
	}
	if strings.TrimSpace(m.Story) == "" {
		t.Fatal("mission story is empty")
	}
	if len(m.Hints) == 0 {
		t.Fatal("mission hints are empty")
	}
	if strings.TrimSpace(m.Answer) == "" {
		t.Fatal("mission answer is empty")
	}
	if strings.TrimSpace(tmpl) == "" {
		t.Fatal("mission template is empty")
	}
	if !strings.Contains(tmpl, "// === YOUR CODE HERE ===") || !strings.Contains(tmpl, "// === END ===") {
		t.Fatal("mission template is missing player code markers")
	}
	if !m.Check.StdoutNonempty && m.Check.StdoutContains == "" && m.Check.StdoutEquals == "" {
		t.Fatal("mission check has no assertion")
	}
}
