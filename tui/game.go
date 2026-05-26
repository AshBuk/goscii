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
	"github.com/charmbracelet/bubbles/viewport"
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
	mission         *levels.Mission
	template        string
	header          string // read-only context shown above the editor
	hdrPort         viewport.Model
	scaffold        string
	answerFormatted string // gofmt result of mission.Answer; empty for prose answers
	answerIsCode    bool   // true when answerFormatted is valid Go
	editor          textarea.Model
	provider        ai.Provider // nil in offline mode
	state           gameState
	lastOutput      string
	analysis        string
	hintIdx         int // -1 = hidden
	showAnswer      bool
	next            bool
	statusCollapsed bool
	step            int // current mission in chain (0 = no chain)
	maxStep         int // total missions in chain
	width           int
	height          int
}

// recalcEditorHeight recomputes the editor height using stored dimensions and
// current collapsed state. Safe to call any time after the first WindowSizeMsg.
func (m *Model) recalcEditorHeight() {
	if m.height == 0 {
		return
	}
	// world height is dynamic: story text wraps at different widths.
	worldH := strings.Count(renderWorld(*m), "\n") + 1
	// \n\n(2) + scaffold(1) + }(1) + \n(1) + indicator(1) = 6
	const restFixed = 6
	statusH := 4
	if m.statusCollapsed {
		statusH = 0
	}
	headerH := 0
	if m.hdrPort.Height > 0 {
		headerH = m.hdrPort.Height + 1
	}
	if h := m.height - worldH - restFixed - statusH - headerH; h >= 3 {
		m.editor.SetHeight(h)
	}
}

// Passed reports whether the player completed the level successfully.
func (m Model) Passed() bool { return m.state == statePassed }

// NextRequested reports whether the player asked for another mission.
func (m Model) NextRequested() bool { return m.next }

// New creates a game model. provider may be nil (offline / onboarding mode).
// step and maxStep track chain progress; pass 0 for both when there is no chain.
func New(m *levels.Mission, tmpl string, provider ai.Provider, step, maxStep int) Model {
	formatted, isCode := formatGoSnippet(m.Answer)
	hdr := engine.TemplateHeader(tmpl)
	hdrPort := viewport.New(0, 0)
	hdrPort.SetContent(hdr)
	return Model{
		mission:         m,
		template:        tmpl,
		header:          hdr,
		hdrPort:         hdrPort,
		scaffold:        engine.ScaffoldAfter(tmpl),
		answerFormatted: formatted,
		answerIsCode:    isCode,
		editor:          newEditor(),
		provider:        provider,
		state:           stateIdle,
		hintIdx:         -1,
		step:            step,
		maxStep:         maxStep,
	}
}

// PlayerCode returns the code the player submitted when the mission passed.
func (m Model) PlayerCode() string {
	if m.state == statePassed {
		return m.editor.Value()
	}
	return ""
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocyclo // bubbletea "one switch" updates
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
			if m.mission.Difficulty != levels.Survival {
				if n := len(m.mission.Hints); n > 0 {
					m.hintIdx = (m.hintIdx + 1) % n
				}
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
		case "enter":
			lines := strings.Split(m.editor.Value(), "\n")
			lineNum := m.editor.Line()
			indent := ""
			if lineNum < len(lines) {
				line := lines[lineNum]
				trimLeft := strings.TrimLeft(line, " ")
				indent = line[:len(line)-len(trimLeft)]
				if strings.HasSuffix(strings.TrimRight(line, " "), "{") {
					indent += "    "
				}
			}
			m.editor.InsertString("\n" + indent)
			return m, nil
		case "ctrl+s":
			if formatted, ok := formatGoSnippet(m.editor.Value()); ok {
				m.editor.SetValue(formatted)
			}
			return m, nil
		case "tab":
			m.editor.InsertString("    ")
			return m, nil
		case "{":
			m.editor.InsertString("{}")
			return m, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "(":
			m.editor.InsertString("()")
			return m, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "[":
			m.editor.InsertString("[]")
			return m, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "ctrl+e":
			if m.state == stateFailed {
				if target := parseFirstErrorLine(m.lastOutput); target > 0 {
					return m, jumpCursorCmd(m.editor.Line(), target-1)
				}
			}
			return m, nil
		case "alt+up":
			m.hdrPort.ScrollUp(1)
			return m, nil
		case "alt+down":
			m.hdrPort.ScrollDown(1)
			return m, nil
		case "ctrl+b":
			m.statusCollapsed = !m.statusCollapsed
			m.recalcEditorHeight()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetWidth(msg.Width - 4)
		if m.header != "" {
			const maxHeaderLines = 6
			hdrH := min(strings.Count(m.header, "\n")+1, maxHeaderLines)
			m.hdrPort.Width = msg.Width - 4
			m.hdrPort.Height = hdrH
		}
		m.recalcEditorHeight()

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
	if m.hdrPort.Height > 0 {
		sb.WriteString(templateHeaderStyle.Render(m.hdrPort.View()))
		sb.WriteString("\n")
	}
	sb.WriteString(m.editor.View())
	if m.scaffold != "" {
		sb.WriteString("\n")
		sb.WriteString(scaffoldStyle.Render("GOSCII ▸ " + m.scaffold))
	}
	if m.header != "" {
		sb.WriteString("\n")
		sb.WriteString(templateHeaderStyle.Render("}"))
	}
	sb.WriteString("\n")
	if m.statusCollapsed {
		sb.WriteString(keysStyle.Render("[ctrl+b] ▶"))
		switch m.state {
		case statePassed:
			sb.WriteString(" ")
			sb.WriteString(passStyle.Render("PASSED"))
		case stateFailed:
			sb.WriteString(" ")
			sb.WriteString(failStyle.Render("FAILED"))
		case stateRunning:
			sb.WriteString(" running...")
		case stateAnalyzing:
			sb.WriteString(" ")
			sb.WriteString(analysisStyle.Render("analyzing..."))
		}
	} else {
		sb.WriteString(keysStyle.Render("[ctrl+b] ▼"))
		sb.WriteString("\n")
		if signal := renderSignalContract(m); signal != "" {
			sb.WriteString(signal)
			sb.WriteString("\n")
		}
		sb.WriteString(renderStatus(m))
	}
	return sb.String()
}

var (
	templateHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	passStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	failStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	analysisStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Italic(true)
	keysStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	scaffoldStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	contractStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
	contractValStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
)

// renderSignalContract shows the expected output to the player on all difficulties except survival.
func renderSignalContract(m Model) string {
	if m.mission.Difficulty == levels.Survival {
		return ""
	}
	check := m.mission.Check
	switch {
	case check.StdoutEquals != "":
		return contractStyle.Render("SIGNAL ▸ target output → ") + contractValStyle.Render(check.StdoutEquals)
	case check.StdoutContains != "":
		return contractStyle.Render("SIGNAL ▸ output must contain → ") + contractValStyle.Render(check.StdoutContains)
	case check.StdoutNonempty:
		return contractStyle.Render("SIGNAL ▸ any output accepted")
	default:
		return ""
	}
}

func renderStatus(m Model) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	lines := renderCockpitLines(m, w)
	if hints := m.mission.Hints; m.hintIdx >= 0 && m.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", m.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w-2).Render("GOSCII ["+counter+"] "+hints[m.hintIdx]))
	}
	if m.showAnswer {
		if m.answerIsCode {
			lines = append(lines, answerStyle.Render("GOSCII [answer]"))
			lines = append(lines, answerStyle.Width(w-2).Render(m.answerFormatted))
		} else {
			lines = append(lines, answerStyle.Width(w-2).Render("GOSCII [answer] "+m.mission.Answer))
		}
	}
	if m.state == stateFailed || m.state == stateIdle {
		lines = append(lines, keysStyle.Render(renderKeyBar(m)))
	}
	return strings.Join(lines, "\n")
}

func renderCockpitLines(m Model, w int) []string {
	switch m.state {
	case stateRunning:
		return []string{"running..."}
	case stateAnalyzing:
		return []string{analysisStyle.Render("GOSCII is reading the logs...")}
	case statePassed:
		lines := []string{passStyle.Render("PASSED")}
		if m.lastOutput != "" {
			lines = append(lines, outputStyle.Render(">> "+m.lastOutput))
		}
		return append(lines, keysStyle.Render("[ctrl+n] next   [ctrl+c] quit"))
	case stateFailed:
		lines := []string{failStyle.Width(w - 2).Render(m.lastOutput)}
		if m.analysis != "" {
			lines = append(lines, analysisStyle.Width(w-2).Render("GOSCII ▸ "+m.analysis))
		}
		return lines
	}
	return nil
}

func renderKeyBar(m Model) string {
	keys := "[ctrl+r] run   [ctrl+s] fmt"
	if m.mission.Difficulty != levels.Survival {
		keys += "   [ctrl+h] hint"
	}
	if m.provider != nil {
		keys += "   [ctrl+g] analyze"
	}
	if m.mission.Answer != "" {
		keys += "   [ctrl+a] answer"
	}
	if m.state == stateFailed {
		if n := countErrorLines(m.lastOutput); n > 0 {
			errLabel := "[ctrl+e] goto error"
			if n > 1 {
				errLabel += fmt.Sprintf(" (%d)", n)
			}
			keys += "   " + errLabel
		}
	}
	return keys + "   [ctrl+c] quit"
}
