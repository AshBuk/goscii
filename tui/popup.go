// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Mission and Difficulty are modal popups layered over the live hub menu — they
// are HubModel state (not separate screens), so their update and render logic
// lives here rather than in a child model.
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/levels"
)

type hubPopup int

const (
	popupNone hubPopup = iota
	popupDifficulty
	popupMission
)

func (h HubModel) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if h.diffCursor > 0 {
			h.diffCursor--
		}
	case "down", "j":
		if h.diffCursor < len(levels.Difficulties)-1 {
			h.diffCursor++
		}
	case "enter":
		h.difficulty = levels.Difficulties[h.diffCursor]
		h.popup = popupNone
	case "esc":
		h.popup = popupNone
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) updateMission(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if h.topicCursor > 0 {
			h.topicCursor--
		}
	case "down", "j":
		if h.topicCursor < len(ai.StdlibTopics)-1 {
			h.topicCursor++
		}
	case "enter":
		t := ai.StdlibTopics[h.topicCursor]
		h.popup = popupNone
		return h, func() tea.Msg { return TopicSelectedMsg{Topic: t} }
	case "esc":
		h.popup = popupNone
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

// popupBox renders the active popup's framed content (difficulty or mission).
func (h HubModel) popupBox() string {
	var title string
	var rows []string
	switch h.popup {
	case popupDifficulty:
		title = "DIFFICULTY"
		for i, d := range levels.Difficulties {
			rows = append(rows, popupRow(string(d), "", i == h.diffCursor))
		}
	case popupMission:
		title = "MISSION"
		for i, t := range ai.StdlibTopics {
			badge := ""
			if n := len(h.progress.Topics[t.Slug].Completed); n > 0 {
				badge = fmt.Sprintf("  [%d]", n)
			}
			rows = append(rows, popupRow(t.Title, badge, i == h.topicCursor))
		}
	}
	lines := append([]string{styleAccent.Render(title), ""}, rows...)
	lines = append(lines, "", styleHint.Render("[↑/↓] select   [enter] confirm   [esc] back"))
	return menuBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// popupRow renders one popup list line: a cursor caret, the name, and an
// optional dim badge.
func popupRow(name, badge string, active bool) string {
	cursor := "  "
	style := styleMuted
	if active {
		cursor = "> "
		style = styleText
	}
	return style.Render(cursor+name) + styleDim.Render(badge)
}

// overlayCenter composites popup centered over base via the lipgloss layer
// compositor (the same mechanism Crush uses for its modals).
func overlayCenter(base, popup string, w, h int) string {
	x := max(0, (w-lipgloss.Width(popup))/2)
	y := max(0, (h-lipgloss.Height(popup))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(popup).X(x).Y(y).Z(1),
	).Render()
}
