// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package levels

import (
	"strings"
	"testing"

	"github.com/AshBuk/goscii/engine"
)

func TestBundledMissionsAreLoadable(t *testing.T) {
	for _, a := range BundledAdventures {
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
	if !m.Check.StdoutNonempty && m.Check.StdoutContains == "" && m.Check.StdoutEquals == "" {
		t.Fatal("mission check has no assertion")
	}
	assertSolvable(t, m, tmpl)
}

// assertSolvable runs the reference against the mission's own check.
func assertSolvable(t *testing.T, m *Mission, tmpl string) {
	t.Helper()
	if m.Check.Prediction {
		r := engine.RunCode(tmpl, "")
		if !r.ExitOK || !engine.SameOutput(r.Stdout, m.Check.StdoutEquals) {
			t.Fatalf("decode stdout_equals must match a real run, got:\n%s%s", r.Stdout, r.Stderr)
		}
		if v := engine.Judge(tmpl, m.Answer, m.Check); !v.Passed {
			t.Fatalf("decode answer must match stdout_equals: %s", v.Failure)
		}
		return
	}
	if !strings.Contains(tmpl, "// === YOUR CODE HERE ===") || !strings.Contains(tmpl, "// === END ===") {
		t.Fatal("mission template is missing player code markers")
	}
	if m.Check.StdoutNonempty {
		return // open beat: the answer is prose, not code
	}
	if v := engine.Judge(tmpl, m.Answer, m.Check); !v.Passed {
		t.Fatalf("reference answer fails its own check: %s %s", v.Failure, v.Result.Stderr)
	}
	if m.Seed != "" {
		if v := engine.Judge(tmpl, m.Seed, m.Check); v.Passed {
			t.Fatal("the seed must not pass as it is")
		}
	}
}

func TestCrewLogsAreCanon(t *testing.T) {
	logs := CrewLogs()
	if len(logs) < 10 {
		t.Fatalf("want a full crew log, got %d fragments", len(logs))
	}
	seen := map[string]bool{}
	for _, l := range logs {
		if l.ID == "" || l.Fact == "" || seen[l.ID] {
			t.Fatalf("bad fragment %+v", l)
		}
		seen[l.ID] = true
	}
}

func TestResumeSurvivesRenumbering(t *testing.T) {
	paths := []string{"adventures/x/01_variables", "adventures/x/02_types_decode", "adventures/x/03_types"}
	cases := map[string]int{
		"":                             0,
		"adventures/x/03_types":        2,
		"adventures/x/02_types":        2,
		"adventures/x/07_gone":         0,
		"adventures/x/02_types_decode": 1,
	}
	for cp, want := range cases {
		if got := Resume(paths, cp); got != want {
			t.Errorf("Resume(%q) = %d, want %d", cp, got, want)
		}
	}
}
