// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Navigator scans compiler output for line numbers and moves the textarea cursor to the first error.
package tui

import (
	"regexp"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// editorErrRe matches normalized error lines produced by engine.NormalizeErrors.
var editorErrRe = regexp.MustCompile(`(?m)^line (\d+):`)

// parseFirstErrorLine returns the 1-based editor line of the first compiler
// error in output, or 0 if none found.
func parseFirstErrorLine(output string) int {
	sub := editorErrRe.FindStringSubmatch(output)
	if len(sub) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(sub[1])
	return n
}

// countErrorLines counts distinct "line N:" occurrences in output.
func countErrorLines(output string) int {
	return len(editorErrRe.FindAllString(output, -1))
}

// jumpCursorCmd returns a command that moves the textarea cursor from current
// to target (both 0-indexed) by injecting synthetic up/down key messages.
func jumpCursorCmd(current, target int) tea.Cmd {
	delta := target - current
	if delta == 0 {
		return nil
	}
	keyType := tea.KeyDown
	if delta < 0 {
		keyType = tea.KeyUp
		delta = -delta
	}
	cmds := make([]tea.Cmd, delta)
	for i := range cmds {
		cmds[i] = func() tea.Msg { return tea.KeyMsg{Type: keyType} }
	}
	return tea.Sequence(cmds...)
}
