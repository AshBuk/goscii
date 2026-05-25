// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package tui renders the terminal cockpit using the Bubble Tea framework.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

type gameState int

const (
	stateIdle gameState = iota
	stateRunning
	statePassed
	stateFailed
	stateAnalyzing // GOSCII is reading the logs
)

type runDoneMsg engine.RunResult
type analysisDoneMsg struct{ text string }
type analysisErrMsg struct{ err error }

// Model is the main game screen.
type Model struct {
	mission    *levels.Mission
	template   string
	scaffold   string
	editor     textarea.Model
	provider   ai.Provider // nil in offline mode
	state      gameState
	lastOutput string
	analysis   string
	hintIdx    int // -1 = hidden
	showAnswer bool
	next       bool
	width      int
	height     int
}

// Passed reports whether the player completed the level successfully.
func (m Model) Passed() bool { return m.state == statePassed }

// NextRequested reports whether the player asked for another mission.
func (m Model) NextRequested() bool { return m.next }

// New creates a game model. provider may be nil (offline / onboarding mode).
func New(m *levels.Mission, tmpl string, provider ai.Provider) Model {
	return Model{
		mission:  m,
		template: tmpl,
		scaffold: engine.ScaffoldAfter(tmpl),
		editor:   newEditor(),
		provider: provider,
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

func (m Model) analyzeLogs() tea.Cmd {
	if m.provider == nil {
		return nil
	}
	req := ai.AnalyzeRequest{
		Concept: m.mission.Concept,
		Code:    m.editor.Value(),
		ErrMsg:  m.lastOutput,
	}
	return func() tea.Msg {
		text, err := m.provider.Analyze(context.Background(), req)
		if err != nil {
			return analysisErrMsg{err}
		}
		return analysisDoneMsg{text}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if m.state != stateRunning && m.state != stateAnalyzing {
				m.state = stateRunning
				m.analysis = ""
				return m, m.runCode()
			}
		case "ctrl+h":
			if n := len(m.mission.Hints); n > 0 {
				m.hintIdx = (m.hintIdx + 1) % n
			}
			return m, nil
		case "ctrl+a":
			if m.mission.Answer != "" {
				m.showAnswer = !m.showAnswer
			}
			return m, nil
		case "ctrl+g":
			// GOSCII Logs Analyzer — only available on failure, only with a provider
			if m.state == stateFailed && m.provider != nil {
				m.state = stateAnalyzing
				m.analysis = ""
				return m, m.analyzeLogs()
			}
		case "ctrl+n":
			if m.state == statePassed {
				m.next = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetWidth(msg.Width - 4)
		// Reserve lines for: header(1) + blank(1) + terrain(1) + sprite(4) + blank(1) +
		// story(3) + blank(2) + scaffold(1) + blank(1) + status(4) = ~19 lines overhead.
		if editorH := msg.Height - 19; editorH >= 3 {
			m.editor.SetHeight(editorH)
		}

	case runDoneMsg:
		result := engine.RunResult(msg)
		if m.mission.Check.Verify(result) {
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

	case analysisDoneMsg:
		m.state = stateFailed // back to failed so player can keep editing
		m.analysis = msg.text
		return m, nil

	case analysisErrMsg:
		m.state = stateFailed
		m.analysis = "GOSCII signal lost. " + msg.err.Error()
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
	analysisStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Italic(true)
	keysStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	scaffoldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	outputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
)

func renderStatus(m Model) string {
	var lines []string

	w := m.width
	if w <= 0 {
		w = 80
	}

	switch m.state {
	case stateRunning:
		lines = append(lines, "running...")

	case stateAnalyzing:
		lines = append(lines, analysisStyle.Render("GOSCII is reading the logs..."))

	case statePassed:
		lines = append(lines, passStyle.Render("PASSED"))
		if m.lastOutput != "" {
			lines = append(lines, outputStyle.Render(">> "+m.lastOutput))
		}
		lines = append(lines, keysStyle.Render("[ctrl+n] next   [ctrl+c] quit"))

	case stateFailed:
		lines = append(lines, failStyle.Width(w-2).Render(m.lastOutput))
		if m.analysis != "" {
			lines = append(lines, analysisStyle.Width(w-2).Render("GOSCII ▸ "+m.analysis))
		}
	}

	if hints := m.mission.Hints; m.hintIdx >= 0 && m.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", m.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w-2).Render("GOSCII ["+counter+"] "+hints[m.hintIdx]))
	}

	if m.showAnswer {
		lines = append(lines, answerStyle.Width(w-2).Render("GOSCII [answer] "+m.mission.Answer))
	}

	if m.state == stateFailed || m.state == stateIdle {
		keys := "[ctrl+r] run   [ctrl+h] hint"
		if m.provider != nil {
			keys += "   [ctrl+g] analyze"
		}
		if m.mission.Answer != "" {
			keys += "   [ctrl+a] answer"
		}
		keys += "   [ctrl+c] quit"
		lines = append(lines, keysStyle.Render(keys))
	}

	return strings.Join(lines, "\n")
}
