// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Selector lists available topics and emits TopicSelectedMsg when the player locks a target.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

// SelectorModel is the topic selector shown when the player picks Mission from the hub.
type SelectorModel struct {
	topics      []ai.Topic
	topicCursor int
	stats       map[string]engine.TopicStat
	width       int
}

func NewSelector(stats map[string]engine.TopicStat) SelectorModel {
	if stats == nil {
		stats = make(map[string]engine.TopicStat)
	}
	return SelectorModel{
		topics: ai.StdlibTopics,
		stats:  stats,
	}
}

func (m SelectorModel) Init() tea.Cmd { return nil }

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, func() tea.Msg { return BackMsg{} }
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m SelectorModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.topicCursor > 0 {
			m.topicCursor--
		}
	case "down", "j":
		if m.topicCursor < len(m.topics)-1 {
			m.topicCursor++
		}
	case "enter":
		t := m.topics[m.topicCursor]
		return m, func() tea.Msg { return TopicSelectedMsg{Topic: t} }
	}
	return m, nil
}

func (m SelectorModel) View() string {
	var lines []string
	lines = append(lines, selectorAccent.Render("GOSCII · MISSION PROTOCOL"), "")

	for i, t := range m.topics {
		stat := m.stats[t.Slug]
		count := len(stat.Completed)

		cursor := "  "
		nameStyle := selectorMuted
		if i == m.topicCursor {
			cursor = "> "
			nameStyle = selectorText
		}

		badge := ""
		if count > 0 {
			badge = selectorDim.Render(fmt.Sprintf("  [%d]", count))
		}

		lines = append(lines, nameStyle.Render(cursor+t.Title)+badge)
		if i == m.topicCursor {
			lines = append(lines, selectorMuted.Render("   "+t.Concepts))
		}
	}

	lines = append(lines, "", selectorKeys.Render("[↑/↓ or k/j] navigate   [enter] select   [ctrl+c] back"))
	return selectorFrame(m.width, strings.Join(lines, "\n"))
}

func selectorFrame(width int, body string) string {
	if width < 40 {
		width = 82
	}
	return lipgloss.NewStyle().Width(width-2).Padding(1, 2).Render(body)
}

var (
	selectorAccent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	selectorText   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectorMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	selectorDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	selectorKeys   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
