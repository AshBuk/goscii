// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// StationModel shows the wreck: subsystem integrity and the recovered crew logs.
type StationModel struct {
	progress *engine.Progress
	port     viewport.Model
	width    int
	height   int
}

func NewStation(p *engine.Progress) StationModel {
	if p == nil {
		p = &engine.Progress{}
	}
	return StationModel{progress: p, port: viewport.New()}
}

func (m StationModel) Init() tea.Cmd { return nil }

func (m StationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.port.SetWidth(contentWidth(m.width))
		m.port.SetHeight(max(5, m.height-8))
		m.port.SetContent(m.body())
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.port, cmd = m.port.Update(msg)
	return m, cmd
}

func (m StationModel) body() string {
	cw := contentWidth(m.width)
	st := m.progress.Station
	rows := make([]string, 0, len(engine.Systems))
	for _, sys := range engine.Systems {
		rows = append(rows, styleText.Render(fmt.Sprintf("%-16s", sys.Name))+" "+integrityBar(st.Level(sys.Slug))+styleDim.Render("  "+sys.Brief))
	}
	box := menuBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	lines := []string{
		centerBlock(styleMuted.Render(fmt.Sprintf("Hull-wide integrity %d%% · weakest: %s", st.Overall(), st.Weakest().Name)), cw),
		"",
		centerBlock(box, cw),
		"",
		styleAccent.Render(fmt.Sprintf("CREW LOGS · %d of %d recovered", len(m.progress.Logs), len(levels.CrewLogs()))),
		"",
	}
	if len(m.progress.Logs) == 0 {
		lines = append(lines, styleDim.Render("Complete a mission chain to decrypt a damaged sector."))
	}
	for _, l := range m.progress.Logs {
		lines = append(lines, styleMuted.Render("▸ "+l.ID), storyStyle.Width(cw).Render(l.Text), "")
	}
	return strings.Join(lines, "\n")
}

func (m StationModel) View() tea.View {
	cw := contentWidth(m.width)
	lines := []string{
		centerBlock(styleAccent.Render("GOSCII · STATION"), cw),
		"",
		m.port.View(),
		"",
		keyHints(cw, "[↑/↓] scroll   [esc] back"),
	}
	return tea.NewView(screenFrame(m.width, strings.Join(lines, "\n")))
}
