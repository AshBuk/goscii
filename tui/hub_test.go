// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

func TestHubRestoresDifficulty(t *testing.T) {
	for stored, want := range map[string]levels.Difficulty{
		"hard":  levels.Hard,
		"":      levels.Easy,
		"bogus": levels.Easy,
	} {
		h := NewHub(nil, &engine.Progress{Difficulty: stored})
		if h.difficulty != want {
			t.Errorf("stored %q: difficulty = %q, want %q", stored, h.difficulty, want)
		}
	}
}

func TestHubSavesPickedDifficulty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	h := NewHub(nil, nil)
	h.popup = popupDifficulty
	h.diffCursor = diffIndex(levels.Survival)

	_, cmd := h.updateDiff(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("picking a difficulty returned no save command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("save failed: %v", msg)
	}
	p, err := engine.LoadProgress()
	if err != nil {
		t.Fatal(err)
	}
	if p.Difficulty != string(levels.Survival) {
		t.Errorf("saved difficulty = %q, want %q", p.Difficulty, levels.Survival)
	}
}
