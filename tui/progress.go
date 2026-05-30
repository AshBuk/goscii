// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

const (
	progressBarWidth = 50                    // bar width in cells
	progressBarMax   = 100                   // completions that fill the bar
	progressTick     = 50 * time.Millisecond // one bar cell fills per tick
)

// barFill maps a completion count to filled cells (0..progressBarWidth), scaled
// so the bar is full at progressBarMax.
func barFill(count int) int {
	if count <= 0 {
		return 0
	}
	if count >= progressBarMax {
		return progressBarWidth
	}
	return (count*progressBarWidth + progressBarMax/2) / progressBarMax // rounded
}

type progressTickMsg struct{}

func progressTickCmd() tea.Cmd {
	return tea.Tick(progressTick, func(time.Time) tea.Msg { return progressTickMsg{} })
}

// ProgressModel displays per-topic mission completion stats.
type ProgressModel struct {
	progress *engine.Progress
	width    int
	anim     int
	fullBar  int
}

func NewProgress(p *engine.Progress) ProgressModel {
	if p == nil {
		p = &engine.Progress{Topics: make(map[string]engine.TopicStat)}
	}
	m := ProgressModel{progress: p}
	for _, t := range ai.StdlibTopics {
		if f := barFill(len(p.Topics[t.Slug].Completed)); f > m.fullBar {
			m.fullBar = f
		}
	}
	return m
}

func (m ProgressModel) Init() tea.Cmd {
	if m.fullBar == 0 {
		return nil // nothing completed yet — no fill to animate
	}
	return progressTickCmd()
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case progressTickMsg:
		if m.anim < m.fullBar {
			m.anim++
			if m.anim < m.fullBar {
				return m, progressTickCmd()
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProgressModel) View() tea.View {
	cw := contentWidth(m.width)

	// Width of the name column = widest topic title, for aligned bars.
	nameW := lipgloss.Width("Total")
	for _, t := range ai.StdlibTopics {
		if w := lipgloss.Width(t.Title); w > nameW {
			nameW = w
		}
	}

	total := 0
	rows := make([]string, 0, len(ai.StdlibTopics)+2)
	for _, t := range ai.StdlibTopics {
		count := len(m.progress.Topics[t.Slug].Completed)
		total += count

		fill := min(barFill(count), m.anim)
		bar := styleAccent.Render(strings.Repeat("█", fill)) +
			styleDim.Render(strings.Repeat("░", progressBarWidth-fill))
		cnt := styleMuted.Render("·")
		if count > 0 {
			cnt = styleAccent.Render(fmt.Sprintf("%d", count))
		}
		name := styleText.Render(fmt.Sprintf("%-*s", nameW, t.Title))
		rows = append(rows, name+"  "+bar+"  "+cnt)
	}

	innerW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r); w > innerW {
			innerW = w
		}
	}
	rows = append(rows,
		styleMuted.Render(strings.Repeat("─", innerW)),
		styleText.Render(fmt.Sprintf("%-*s", nameW, "Total"))+"  "+styleAccent.Render(fmt.Sprintf("%d completed", total)),
	)
	box := menuBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	lines := []string{
		centerBlock(styleAccent.Render("GOSCII · PROGRESS LOG"), cw),
		"",
		centerBlock(box, cw),
		"",
		keyHints(cw, "[esc] back"),
	}
	return tea.NewView(screenFrame(m.width, strings.Join(lines, "\n")))
}
