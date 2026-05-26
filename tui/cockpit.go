// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
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
