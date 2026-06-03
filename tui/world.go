// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/assets"
)

// worldContent is the scrollable body of the world pane: the GOSCII banner, the
// topic art, and the mission story. The [ctrl+t] toggle is chrome drawn by the
// cockpit, not part of this content.
func worldContent(c Cockpit, w int) string {
	title := fmt.Sprintf("GOSCII  |  %s", c.mission.Title)
	if c.maxStep > 0 {
		title += fmt.Sprintf("  [%d/%d]", c.step, c.maxStep)
	}
	goscii := "Go Orbital Survival Coding Interactive Interface"
	gap := 2
	if space := c.width - len(title) - len(goscii); space > 2 {
		gap = space
	}
	header := headerStyle.Render(title) + strings.Repeat(" ", gap) + gosciiStyle.Render(goscii)

	art := spriteStyle(c.state).Render(artFor(c.topic))
	if w < 40 {
		w = 76
	}
	story := storyStyle.Width(w).Render(strings.TrimRight(c.mission.Story, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", art, "", story)
}

// spriteStyle colors the topic art
func spriteStyle(s cockpitState) lipgloss.Style {
	switch s {
	case statePassed:
		return lipgloss.NewStyle().Foreground(colorPass)
	case stateFailed:
		return lipgloss.NewStyle().Foreground(colorFail)
	default:
		return lipgloss.NewStyle().Foreground(colorAccent)
	}
}

// artFor returns the box-drawing art for a topic slug
func artFor(topic string) string {
	data, err := assets.FS.ReadFile("art/" + topic + ".txt")
	if err != nil {
		data, _ = assets.FS.ReadFile("art/default.txt") // always embedded
	}
	return strings.TrimRight(string(data), "\n")
}
