// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package tui renders the terminal cockpit using the Bubble Tea framework.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

type gameState int

const (
	stateIdle gameState = iota
	stateRunning
	statePassed
	stateFailed
)

type runDoneMsg engine.RunResult

type Model struct {
	level      *levels.Level
	template   string
	scaffold   string
	editor     textarea.Model
	state      gameState
	lastOutput string
	hintIdx    int // -1 = hidden
	showAnswer bool
	width      int
	height     int
}

func New(l *levels.Level, tmpl string) Model {
	return Model{
		level:    l,
		template: tmpl,
		scaffold: engine.ScaffoldAfter(tmpl),
		editor:   newEditor(),
		state:    stateIdle,
		hintIdx:  -1,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) runCode() tea.Cmd {
	code := m.editor.Value()
	tmpl := m.template
	return func() tea.Msg {
		return runDoneMsg(engine.RunCode(tmpl, code))
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if m.state != stateRunning {
				m.state = stateRunning
				return m, m.runCode()
			}
		case "ctrl+h":
			if n := len(m.level.Hints); n > 0 {
				m.hintIdx = (m.hintIdx + 1) % n
			}
			return m, nil
		case "ctrl+a":
			if m.level.Answer != "" {
				m.showAnswer = !m.showAnswer
			}
			return m, nil
		case "ctrl+n":
			if m.state == statePassed {
				// TODO: advance to next level
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetWidth(msg.Width - 4)

	case runDoneMsg:
		result := engine.RunResult(msg)
		if m.level.Check.Verify(result) {
			m.state = statePassed
			m.lastOutput = strings.TrimSpace(result.Stdout)
		} else {
			m.state = stateFailed
			if result.Stderr != "" {
				m.lastOutput = result.Stderr
			} else {
				m.lastOutput = fmt.Sprintf("got: %q", strings.TrimSpace(result.Stdout))
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(renderWorld(m))
	sb.WriteString("\n\n")
	sb.WriteString(m.editor.View())
	if m.scaffold != "" {
		sb.WriteString("\n")
		sb.WriteString(scaffoldStyle.Render("GOSCII ▸ " + m.scaffold))
	}
	sb.WriteString("\n")
	sb.WriteString(renderStatus(m))
	return sb.String()
}

var (
	passStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	failStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	keysStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	scaffoldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	outputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
)

func renderStatus(m Model) string {
	var lines []string

	switch m.state {
	case stateRunning:
		lines = append(lines, "running...")
	case statePassed:
		lines = append(lines, passStyle.Render("PASSED"))
		if m.lastOutput != "" {
			lines = append(lines, outputStyle.Render(">> "+m.lastOutput))
		}
		lines = append(lines, keysStyle.Render("[ctrl+n] next level   [ctrl+c] quit"))
	case stateFailed:
		lines = append(lines, failStyle.Render(m.lastOutput))
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	if hints := m.level.Hints; m.hintIdx >= 0 && m.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", m.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w-2).Render("GOSCII ["+counter+"] "+hints[m.hintIdx]))
	}

	if m.showAnswer {
		lines = append(lines, answerStyle.Width(w-2).Render("GOSCII [answer] "+m.level.Answer))
	}

	if m.state != statePassed {
		keys := "[ctrl+r] run   [ctrl+h] hint"
		if m.level.Answer != "" {
			keys += "   [ctrl+a] answer"
		}
		keys += "   [ctrl+c] quit"
		lines = append(lines, keysStyle.Render(keys))
	}

	return strings.Join(lines, "\n")
}
