// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"
	"testing"

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
