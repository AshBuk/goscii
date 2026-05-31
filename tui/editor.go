// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func newEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "// your input"
	ta.ShowLineNumbers = true
	ta.Focus()
	ta.SetVirtualCursor(false) // use the real cursor: terminal blinks it without repainting
	st := ta.Styles()
	st.Cursor.Shape = tea.CursorBar // thin vertical bar instead of the default block
	ta.SetStyles(st)
	ta.KeyMap.WordForward = key.NewBinding(key.WithKeys("ctrl+right"))
	ta.KeyMap.WordBackward = key.NewBinding(key.WithKeys("ctrl+left"))
	ta.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("ctrl+backspace", "alt+backspace", "ctrl+w"))
	return ta
}

// editorTopRow returns the 0-based screen row of the editor's first line: the
// rows above it (world + blank separator + header). Shared by cursor placement
// and the height budget so they can't drift apart.
func editorTopRow(c Cockpit) int {
	rows := strings.Count(renderWorld(c), "\n") + 1 // world block
	rows++                                          // blank separator ("\n\n")
	if c.hdrPort.Height() > 0 {
		rows += c.hdrPort.Height() // header lines
	}
	return rows
}

// recalcEditorHeight sizes the editor to fill the gap between the rows above it
// and the measured height of the block below it, so the status panel and its
// logs always stay on screen.
func (c *Cockpit) recalcEditorHeight() {
	if c.height == 0 {
		return
	}
	h := c.height - editorTopRow(*c) - lipgloss.Height(belowEditor(*c))
	c.editor.SetHeight(max(3, h))
}

// handleEditorKey implements the editor's typing behaviors: auto-indent on
// enter, gofmt on ctrl+s, soft tabs, and bracket auto-pairing / overtyping.
// Returns handled=false for keys it does not act on, so they reach the textarea.
func (c Cockpit) handleEditorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		lines := strings.Split(c.editor.Value(), "\n")
		lineNum := c.editor.Line()
		indent := ""
		if lineNum < len(lines) {
			line := lines[lineNum]
			trimLeft := strings.TrimLeft(line, " ")
			indent = line[:len(line)-len(trimLeft)]
			if strings.HasSuffix(strings.TrimRight(line, " "), "{") {
				indent += "    "
			}
		}
		c.editor.InsertString("\n" + indent)
		return c, nil, true
	case "ctrl+s":
		if formatted, ok := formatGoSnippet(c.editor.Value()); ok {
			c.editor.SetValue(formatted)
		}
		return c, nil, true
	case "tab":
		c.editor.InsertString("    ")
		return c, nil, true
	case "{":
		c.editor.InsertString("{}")
		return c, func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyLeft} }, true
	case "(":
		c.editor.InsertString("()")
		return c, func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyLeft} }, true
	case "[":
		c.editor.InsertString("[]")
		return c, func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyLeft} }, true
	case ")", "]", "}":
		// Overtype the auto-inserted closer instead of duplicating it.
		if charRightOfCursor(c) == rune(msg.String()[0]) {
			c.editor.SetCursorColumn(c.editor.Column() + 1)
			return c, nil, true
		}
	}
	return c, nil, false
}

// charRightOfCursor returns the rune immediately after the cursor on the
// current line, or 0 when the cursor sits at end of line. Used to overtype an
// auto-inserted closing bracket instead of duplicating it.
func charRightOfCursor(c Cockpit) rune {
	lines := strings.Split(c.editor.Value(), "\n")
	row := c.editor.Line()
	if row < 0 || row >= len(lines) {
		return 0
	}
	runes := []rune(lines[row])
	col := c.editor.Column()
	if col < 0 || col >= len(runes) {
		return 0
	}
	return runes[col]
}

// --- error navigation ---
//
// ctrl+e jumps the cursor to the first error.
// These helpers parse the normalized compiler output produced by
// engine.NormalizeErrors, which formats locations as "line N:col:".

// editorErrRe matches a normalized error line, capturing the 1-based editor
// line and (optionally) the column.
var editorErrRe = regexp.MustCompile(`(?m)^line (\d+):(?:(\d+):)?`)

// parseFirstError returns the 1-based line and column of the first compiler
// error in output. line is 0 when no error is found; col is 0 when the
// compiler reported no column.
func parseFirstError(output string) (line, col int) {
	sub := editorErrRe.FindStringSubmatch(output)
	if len(sub) < 2 {
		return 0, 0
	}
	line, _ = strconv.Atoi(sub[1])
	col, _ = strconv.Atoi(sub[2]) // sub[2] is "" (0)
	return line, col
}

func countErrorLines(output string) int {
	return len(editorErrRe.FindAllString(output, -1))
}
