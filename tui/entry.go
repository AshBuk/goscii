// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

type entryStep int

const (
	stepTopics entryStep = iota
	stepDifficulty
)

// EntryModel is the topic + difficulty selector shown on goscii start.
type EntryModel struct {
	step        entryStep
	topics      []ai.Topic
	topicCursor int
	diffCursor  int
	stats       map[string]engine.TopicStat
	selected    *ai.Selection
	width       int
}

func NewEntry(stats map[string]engine.TopicStat) EntryModel {
	if stats == nil {
		stats = make(map[string]engine.TopicStat)
	}
	return EntryModel{
		topics: ai.StdlibTopics,
		stats:  stats,
	}
}

// Selected returns the player's choice, or nil if they quit.
func (m EntryModel) Selected() *ai.Selection { return m.selected }

func (m EntryModel) Init() tea.Cmd { return nil }

func (m EntryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m EntryModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepTopics:
		return m.handleTopicsKey(msg)
	case stepDifficulty:
		return m.handleDifficultyKey(msg)
	}
	return m, nil
}

func (m EntryModel) handleTopicsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.step = stepDifficulty
		m.diffCursor = 0
	}
	return m, nil
}

func (m EntryModel) handleDifficultyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.diffCursor > 0 {
			m.diffCursor--
		}
	case "down", "j":
		if m.diffCursor < len(ai.Difficulties)-1 {
			m.diffCursor++
		}
	case "enter":
		m.selected = &ai.Selection{
			Topic:      m.topics[m.topicCursor],
			Difficulty: ai.Difficulties[m.diffCursor],
		}
		return m, tea.Quit
	case "esc":
		m.step = stepTopics
	}
	return m, nil
}

func (m EntryModel) View() string {
	switch m.step {
	case stepTopics:
		return m.viewTopics()
	case stepDifficulty:
		return m.viewDifficulty()
	}
	return ""
}

func (m EntryModel) viewTopics() string {
	var lines []string
	lines = append(lines, entryAccent.Render("GOSCII — SELECT MISSION"), "")

	for i, t := range m.topics {
		stat := m.stats[t.Slug]
		count := len(stat.Completed)

		cursor := "  "
		nameStyle := entryMuted
		if i == m.topicCursor {
			cursor = "> "
			nameStyle = entryText
		}

		badge := ""
		if count > 0 {
			badge = entryDim.Render(fmt.Sprintf("  [%d]", count))
		}

		lines = append(lines, nameStyle.Render(cursor+t.Title)+badge)
		if i == m.topicCursor {
			lines = append(lines, entryMuted.Render("   "+t.Concepts))
		}
	}

	lines = append(lines, "", entryKeys.Render("[↑/↓ or k/j] navigate   [enter] select   [ctrl+c] quit"))
	return entryFrame(m.width, strings.Join(lines, "\n"))
}

func (m EntryModel) viewDifficulty() string {
	topic := m.topics[m.topicCursor]
	stat := m.stats[topic.Slug]

	var lines []string
	lines = append(lines, entryAccent.Render("MISSION: "+topic.Title))
	lines = append(lines, entryMuted.Render(topic.Concepts), "")

	if total := len(stat.Completed); total > 0 {
		lines = append(lines, entryDim.Render(fmt.Sprintf("Completed: %d mission(s)", total)), "")
	}

	lines = append(lines, entryText.Render("Select difficulty:"), "")
	for i, d := range ai.Difficulties {
		cursor := "  "
		style := entryMuted
		if i == m.diffCursor {
			cursor = "> "
			style = entryText
		}
		lines = append(lines, style.Render(cursor+string(d)))
	}

	lines = append(lines, "", entryKeys.Render("[↑/↓] select   [enter] generate   [esc] back   [ctrl+c] quit"))
	return entryFrame(m.width, strings.Join(lines, "\n"))
}

func entryFrame(width int, body string) string {
	if width < 40 {
		width = 82
	}
	return lipgloss.NewStyle().Width(width-2).Padding(1, 2).Render(body)
}

var (
	entryAccent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	entryText   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	entryMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	entryDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	entryKeys   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
