// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

const progressBarCap = 10

// ProgressModel displays per-topic mission completion stats.
type ProgressModel struct {
	progress *engine.Progress
	width    int
}

func NewProgress(p *engine.Progress) ProgressModel {
	if p == nil {
		p = &engine.Progress{Topics: make(map[string]engine.TopicStat)}
	}
	return ProgressModel{progress: p}
}

func (m ProgressModel) Init() tea.Cmd { return nil }

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProgressModel) View() string {
	cw := contentWidth(m.width)
	var lines []string
	lines = append(lines, centerBlock(styleAccent.Render("GOSCII · PROGRESS LOG"), cw), "")

	total := 0
	for _, t := range ai.StdlibTopics {
		stat := m.progress.Topics[t.Slug]
		count := len(stat.Completed)
		total += count

		fill := min(count, progressBarCap)
		bar := styleAccent.Render(strings.Repeat("█", fill)) +
			styleDim.Render(strings.Repeat("░", progressBarCap-fill))
		name := styleText.Render(fmt.Sprintf("%-28s", t.Title))
		countStr := styleMuted.Render(fmt.Sprintf("%2d", count))
		lines = append(lines, name+"  "+bar+"  "+countStr)
	}

	lines = append(lines,
		"",
		styleMuted.Render(fmt.Sprintf("  Total: %d missions completed", total)),
		"",
		keyHints(cw, "[esc] back"),
	)
	return screenFrame(m.width, strings.Join(lines, "\n"))
}
