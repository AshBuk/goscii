// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/assets"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	gosciiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	worldStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	storyStyle  = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("245"))
)

func renderWorld(m Model) string {
	title := fmt.Sprintf("GOSCII  |  %s", m.mission.Title)
	if m.maxStep > 0 {
		title += fmt.Sprintf("  [%d/%d]", m.step, m.maxStep)
	}
	goscii := "Go Orbital Survival Coding Interactive Interface"
	gap := 2
	if m.width > 0 {
		if space := m.width - len(title) - len(goscii); space > 2 {
			gap = space
		}
	}
	header := headerStyle.Render(title) + strings.Repeat(" ", gap) + gosciiStyle.Render(goscii)

	terrain := worldStyle.Render("  (crash · oblivion) . . . . . . . . . [home · awareness]")
	sprite := worldStyle.Render(spriteFor(m.state))
	w := m.width - 4
	if w < 40 {
		w = 76
	}
	story := storyStyle.Width(w).Render(strings.TrimRight(m.mission.Story, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", terrain, sprite, "", story)
}

func spriteFor(s gameState) string {
	name := "astronaut/idle.txt"
	switch s {
	case stateRunning:
		name = "astronaut/walking.txt"
	case statePassed:
		name = "astronaut/celebrate.txt"
	case stateFailed:
		name = "astronaut/crash.txt"
	}
	data, err := assets.FS.ReadFile(name)
	if err != nil {
		return fmt.Sprintf("[missing sprite: %s: %v]", name, err)
	}
	return strings.TrimRight(string(data), "\n")
}
