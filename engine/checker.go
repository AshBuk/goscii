// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import "strings"

type CheckRule struct {
	StdoutNonempty bool   `yaml:"stdout_nonempty"`
	StdoutContains string `yaml:"stdout_contains"`
	StdoutEquals   string `yaml:"stdout_equals"`
}

func (c CheckRule) Verify(r RunResult) bool {
	if !r.ExitOK {
		return false
	}
	if c.StdoutNonempty && strings.TrimSpace(r.Stdout) == "" {
		return false
	}
	if c.StdoutContains != "" && !strings.Contains(r.Stdout, c.StdoutContains) {
		return false
	}
	if c.StdoutEquals != "" && strings.TrimSpace(r.Stdout) != strings.TrimSpace(c.StdoutEquals) {
		return false
	}
	return true
}
