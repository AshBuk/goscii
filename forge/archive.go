// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// Record is an archived mission, replayable offline.
type Record struct {
	Mission  *levels.Mission `json:"mission"`
	Template string          `json:"template"`
	Signal   string          `json:"signal"`
	Model    string          `json:"model"`
	ForgedAt time.Time       `json:"forged_at"`
	Solved   bool            `json:"solved,omitempty"`
}

func archiveDir() (string, error) {
	dir, err := engine.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "archive"), nil
}

// Archive stores a forged mission. Errors are returned but safe to ignore:
// the archive is a convenience, never needed to play.
func Archive(r Record) error {
	dir, err := archiveDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, safeName(r.Mission.ID)+".json"), data, 0o600)
}

// MarkSolved flags an archived mission as solved.
func MarkSolved(id string) {
	dir, err := archiveDir()
	if err != nil {
		return
	}
	path := filepath.Join(dir, safeName(id)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var r Record
	if json.Unmarshal(data, &r) != nil {
		return
	}
	r.Solved = true
	_ = Archive(r)
}

// Archived lists stored missions, newest first.
func Archived() ([]Record, error) {
	dir, err := archiveDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var r Record
		if json.Unmarshal(data, &r) == nil && r.Mission != nil {
			out = append(out, r)
		}
	}
	slices.SortFunc(out, func(a, b Record) int { return b.ForgedAt.Compare(a.ForgedAt) })
	return out, nil
}

func safeName(id string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, id)
}
