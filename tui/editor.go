// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
)

func newEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "// your input"
	ta.ShowLineNumbers = true
	ta.Focus()
	ta.KeyMap.WordForward = key.NewBinding(key.WithKeys("ctrl+right"))
	ta.KeyMap.WordBackward = key.NewBinding(key.WithKeys("ctrl+left"))
	ta.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("ctrl+backspace", "alt+backspace", "ctrl+w"))
	return ta
}

// recalcEditorHeight recomputes the editor height using stored dimensions and
// current collapsed state. Safe to call any time after the first WindowSizeMsg.
func (c *Cockpit) recalcEditorHeight() {
	if c.height == 0 {
		return
	}
	// world height is dynamic: story text wraps at different widths.
	worldH := strings.Count(renderWorld(*c), "\n") + 1
	// \n\n(2) + scaffold(1) + }(1) + \n(1) + indicator(1) = 6
	const restFixed = 6
	statusH := 4
	if c.statusCollapsed {
		statusH = 0
	}
	headerH := 0
	if c.hdrPort.Height > 0 {
		headerH = c.hdrPort.Height + 1
	}
	if h := c.height - worldH - restFixed - statusH - headerH; h >= 3 {
		c.editor.SetHeight(h)
	}
}
