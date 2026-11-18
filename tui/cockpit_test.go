// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

func cockpitWithCheck(d levels.Difficulty, check engine.CheckRule) Cockpit {
	return Cockpit{
		mission: &levels.Mission{
			Difficulty: d,
			Check:      check,
		},
	}
}

func TestSignalContractHiddenOnSurvival(t *testing.T) {
	c := cockpitWithCheck(levels.Survival, engine.CheckRule{StdoutEquals: "42"})
	if got := renderSignalContract(c); got != "" {
		t.Fatalf("expected empty on survival, got %q", got)
	}
}

func TestSignalContractStdoutEquals(t *testing.T) {
	for _, d := range []levels.Difficulty{levels.Easy, levels.Medium, levels.Hard} {
		c := cockpitWithCheck(d, engine.CheckRule{StdoutEquals: "42"})
		got := renderSignalContract(c)
		if !strings.Contains(got, "42") {
			t.Fatalf("difficulty %s: expected output value in contract, got %q", d, got)
		}
	}
}

func TestSignalContractStdoutContains(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{StdoutContains: "ok"})
	got := renderSignalContract(c)
	if !strings.Contains(got, "ok") {
		t.Fatalf("expected contains value in contract, got %q", got)
	}
}

func TestSignalContractNonempty(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{StdoutNonempty: true})
	got := renderSignalContract(c)
	if !strings.Contains(got, "any output accepted") {
		t.Fatalf("expected nonempty message, got %q", got)
	}
}

func TestSignalContractEmptyCheck(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{})
	if got := renderSignalContract(c); got != "" {
		t.Fatalf("expected empty for no-op check, got %q", got)
	}
}

func TestSignalContractShowsProtocol(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{
		StdoutEquals: "2",
		Requires:     []string{"range"},
		Forbids:      []string{"for-classic"},
		Cases:        []engine.Case{{Input: "x", Stdout: "3"}},
	})
	got := renderSignalContract(c)
	for _, want := range []string{"must use", "a range loop", "must not use", "a three-clause for loop", "+1 shifted readings"} {
		if !strings.Contains(got, want) {
			t.Errorf("contract missing %q:\n%s", want, got)
		}
	}
}

func TestSignalContractPredictionHidesTarget(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{StdoutEquals: "secret", Prediction: true})
	got := renderSignalContract(c)
	if strings.Contains(got, "secret") || !strings.Contains(got, "predict") {
		t.Fatalf("decode contract must ask for a prediction without the answer, got %q", got)
	}
}

func TestSignalContractShowsOxygen(t *testing.T) {
	c := cockpitWithCheck(levels.Survival, engine.CheckRule{StdoutEquals: "42"})
	o2 := 3
	c.oxygen = &o2
	got := renderSignalContract(c)
	if !strings.Contains(got, "O2 3 runs") || strings.Contains(got, "42") {
		t.Fatalf("survival contract shows oxygen and hides the target, got %q", got)
	}
}

func TestDecodeCockpitShowsWholeProgram(t *testing.T) {
	prog := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(1)\n}\n"
	m := &levels.Mission{Kind: levels.Decode, Answer: "1", Check: engine.CheckRule{StdoutEquals: "1", Prediction: true}}
	c := New(m, prog, nil, 0, 0, "variables")
	if c.header != "" || c.footer != "" || c.answerIsCode {
		t.Fatalf("decode keeps no fixed scaffold: header=%q footer=%q", c.header, c.footer)
	}
	if !strings.Contains(worldContent(c, 76), "fmt.Println(1)") {
		t.Fatal("decode program belongs in the scrollable world pane")
	}
}

func TestDecodeFitsSmallTerminal(t *testing.T) {
	m, tmpl, err := levels.Load("adventures/onboarding/22_interfaces_decode")
	if err != nil {
		t.Fatal(err)
	}
	sized, _ := New(m, tmpl, nil, 0, 0, m.Concept).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if h := lipgloss.Height(sized.(Cockpit).View().Content); h > 24 {
		t.Fatalf("decode screen is %d rows on a 24-row terminal", h)
	}
}

func TestRunFromAnotherMissionIsDropped(t *testing.T) {
	tmpl := "package main\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\n\t// === END ===\n}\n"
	m := &levels.Mission{Check: engine.CheckRule{StdoutEquals: "x"}}
	old, cur := New(m, tmpl, nil, 0, 0, "variables"), New(m, tmpl, nil, 0, 0, "variables")
	pass := engine.Verdict{Passed: true}
	got, _ := cur.Update(runDoneMsg{id: old.work.id, verdict: pass})
	if got.(Cockpit).Passed() {
		t.Fatal("a run started by another cockpit must not pass this one")
	}
	got, _ = cur.Update(runDoneMsg{id: cur.work.id, verdict: pass})
	if !got.(Cockpit).Passed() {
		t.Fatal("the cockpit's own run must count")
	}
}

func TestSeedPrefillsEditor(t *testing.T) {
	tmpl := "package main\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\n\t// === END ===\n}\n"
	m := &levels.Mission{Kind: levels.Fault, Seed: "x := 1\n_ = x", Answer: "x := 2\n_ = x"}
	c := New(m, tmpl, nil, 0, 0, "variables")
	if !strings.Contains(c.editor.Value(), "x := 1") {
		t.Fatalf("editor should start with the seed, got %q", c.editor.Value())
	}
}

func TestHintsOffOnSurvivalAndFinale(t *testing.T) {
	c := cockpitWithCheck(levels.Easy, engine.CheckRule{})
	if c.hintsOff() {
		t.Fatal("hints should be on for a regular easy mission")
	}
	c.final = true
	if !c.hintsOff() {
		t.Fatal("chain finale withholds hints")
	}
	if !cockpitWithCheck(levels.Survival, engine.CheckRule{}).hintsOff() {
		t.Fatal("survival withholds hints")
	}
}
