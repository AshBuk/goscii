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

type cockpitState int

const (
	stateIdle cockpitState = iota
	stateRunning
	statePassed
	stateFailed
	stateAnalyzer // GOSCII is reading the logs
)

type runDoneMsg engine.RunResult
type analyzerDoneMsg struct{ text string }
type analyzerErrMsg struct{ err error }

// Cockpit is the main mission screen.
type Cockpit struct {
	mission         *levels.Mission
	template        string
	header          string // read-only context shown above the editor
	hdrPort         viewport.Model
	scaffold        string
	answerFormatted string // gofmt result of mission.Answer; empty for prose answers
	answerIsCode    bool   // true when answerFormatted is valid Go
	editor          textarea.Model
	signal          ai.Provider // nil in offline mode
	state           cockpitState
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

// Passed reports whether the player completed the level successfully.
func (c Cockpit) Passed() bool { return c.state == statePassed }

// NextRequested reports whether the player asked for another mission.
func (c Cockpit) NextRequested() bool { return c.next }

// New creates a cockpit model. signal may be nil (offline / onboarding mode).
// step and maxStep track chain progress; pass 0 for both when there is no chain.
func New(ms *levels.Mission, tmpl string, signal ai.Provider, step, maxStep int) Cockpit {
	formatted, isCode := formatGoSnippet(ms.Answer)
	hdr := engine.TemplateHeader(tmpl)
	hdrPort := viewport.New(0, 0)
	hdrPort.SetContent(hdr)
	return Cockpit{
		mission:         ms,
		template:        tmpl,
		header:          hdr,
		hdrPort:         hdrPort,
		scaffold:        engine.ScaffoldAfter(tmpl),
		answerFormatted: formatted,
		answerIsCode:    isCode,
		editor:          newEditor(),
		signal:          signal,
		state:           stateIdle,
		hintIdx:         -1,
		step:            step,
		maxStep:         maxStep,
	}
}

// PlayerCode returns the code the player submitted when the mission passed.
func (c Cockpit) PlayerCode() string {
	if c.state == statePassed {
		return c.editor.Value()
	}
	return ""
}

func (c Cockpit) Init() tea.Cmd {
	return textarea.Blink
}

func (c Cockpit) runCode() tea.Cmd {
	code := c.editor.Value()
	tmpl := c.template
	return func() tea.Msg {
		return runDoneMsg(engine.RunCode(tmpl, code))
	}
}

func (c Cockpit) runAnalyzer() tea.Cmd {
	if c.signal == nil {
		return nil
	}
	req := ai.AnalyzeRequest{
		Concept: c.mission.Concept,
		Code:    c.editor.Value(),
		ErrMsg:  c.lastOutput,
	}
	return func() tea.Msg {
		text, err := c.signal.Analyze(context.Background(), req)
		if err != nil {
			return analyzerErrMsg{err}
		}
		return analyzerDoneMsg{text}
	}
}

func (c Cockpit) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocyclo // bubbletea "one switch" updates
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return c, tea.Quit
		case "ctrl+r":
			if c.state != stateRunning && c.state != stateAnalyzer {
				c.state = stateRunning
				c.analysis = ""
				return c, c.runCode()
			}
		case "ctrl+h":
			if c.mission.Difficulty != levels.Survival {
				if n := len(c.mission.Hints); n > 0 {
					c.hintIdx = (c.hintIdx + 1) % n
				}
			}
			return c, nil
		case "ctrl+a":
			if c.mission.Answer != "" {
				c.showAnswer = !c.showAnswer
			}
			return c, nil
		case "ctrl+g":
			// GOSCII Logs Analyzer — only available on failure, only with a provider
			if c.state == stateFailed && c.signal != nil {
				c.state = stateAnalyzer
				c.analysis = ""
				return c, c.runAnalyzer()
			}
		case "ctrl+n":
			if c.state == statePassed {
				c.next = true
				return c, tea.Quit
			}
		case "enter":
			lines := strings.Split(c.editor.Value(), "\n")
			lineNum := c.editor.Line()
			indent := ""
			if lineNum < len(lines) {
				line := lines[lineNum]
				trimLeft := strings.TrimLeft(line, " ")
				indent = line[:len(line)-len(trimLeft)]
				if strings.HasSuffix(strings.TrimRight(line, " "), "{") {
					indent += "    "
				}
			}
			c.editor.InsertString("\n" + indent)
			return c, nil
		case "ctrl+s":
			if formatted, ok := formatGoSnippet(c.editor.Value()); ok {
				c.editor.SetValue(formatted)
			}
			return c, nil
		case "tab":
			c.editor.InsertString("    ")
			return c, nil
		case "{":
			c.editor.InsertString("{}")
			return c, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "(":
			c.editor.InsertString("()")
			return c, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "[":
			c.editor.InsertString("[]")
			return c, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyLeft} }
		case "ctrl+e":
			if c.state == stateFailed {
				if target := parseFirstErrorLine(c.lastOutput); target > 0 {
					return c, jumpCursorCmd(c.editor.Line(), target-1)
				}
			}
			return c, nil
		case "alt+up":
			c.hdrPort.ScrollUp(1)
			return c, nil
		case "alt+down":
			c.hdrPort.ScrollDown(1)
			return c, nil
		case "ctrl+b":
			c.statusCollapsed = !c.statusCollapsed
			c.recalcEditorHeight()
			return c, nil
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.editor.SetWidth(msg.Width - 4)
		if c.header != "" {
			const maxHeaderLines = 6
			hdrH := min(strings.Count(c.header, "\n")+1, maxHeaderLines)
			c.hdrPort.Width = msg.Width - 4
			c.hdrPort.Height = hdrH
		}
		c.recalcEditorHeight()

	case runDoneMsg:
		result := engine.RunResult(msg)
		if c.mission.Check.Verify(result) {
			c.state = statePassed
			c.lastOutput = strings.TrimSpace(result.Stdout)
		} else {
			c.state = stateFailed
			if result.Stderr != "" {
				c.lastOutput = result.Stderr
			} else {
				c.lastOutput = fmt.Sprintf("got: %q", strings.TrimSpace(result.Stdout))
			}
		}
		return c, nil

	case analyzerDoneMsg:
		c.state = stateFailed // back to failed so player can keep editing
		c.analysis = msg.text
		return c, nil

	case analyzerErrMsg:
		c.state = stateFailed
		c.analysis = "GOSCII signal lost. " + msg.err.Error()
		return c, nil
	}

	var cmd tea.Cmd
	c.editor, cmd = c.editor.Update(msg)
	return c, cmd
}

func (c Cockpit) View() string {
	var sb strings.Builder
	sb.WriteString(renderWorld(c))
	sb.WriteString("\n\n")
	if c.hdrPort.Height > 0 {
		sb.WriteString(templateHeaderStyle.Render(c.hdrPort.View()))
		sb.WriteString("\n")
	}
	sb.WriteString(c.editor.View())
	if c.scaffold != "" {
		sb.WriteString("\n")
		sb.WriteString(scaffoldStyle.Render("GOSCII ▸ " + c.scaffold))
	}
	if c.header != "" {
		sb.WriteString("\n")
		sb.WriteString(templateHeaderStyle.Render("}"))
	}
	sb.WriteString("\n")
	if c.statusCollapsed {
		sb.WriteString(keysStyle.Render("[ctrl+b] ▶"))
		switch c.state {
		case statePassed:
			sb.WriteString(" ")
			sb.WriteString(passStyle.Render("PASSED"))
		case stateFailed:
			sb.WriteString(" ")
			sb.WriteString(failStyle.Render("FAILED"))
		case stateRunning:
			sb.WriteString(" running...")
		case stateAnalyzer:
			sb.WriteString(" ")
			sb.WriteString(analyzerStyle.Render("logs analyzer..."))
		}
	} else {
		sb.WriteString(keysStyle.Render("[ctrl+b] ▼"))
		sb.WriteString("\n")
		if signal := renderSignalContract(c); signal != "" {
			sb.WriteString(signal)
			sb.WriteString("\n")
		}
		sb.WriteString(renderStatus(c))
	}
	return sb.String()
}

var (
	templateHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	passStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	failStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	analyzerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Italic(true)
	keysStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	scaffoldStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	contractStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
	contractValStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
)

// renderSignalContract shows the expected output to the player on all difficulties except survival.
func renderSignalContract(c Cockpit) string {
	if c.mission.Difficulty == levels.Survival {
		return ""
	}
	check := c.mission.Check
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

func renderStatus(c Cockpit) string {
	w := c.width
	if w <= 0 {
		w = 80
	}
	lines := renderCockpitLines(c, w)
	if hints := c.mission.Hints; c.hintIdx >= 0 && c.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", c.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w-2).Render("GOSCII ["+counter+"] "+hints[c.hintIdx]))
	}
	if c.showAnswer {
		if c.answerIsCode {
			lines = append(lines, answerStyle.Render("GOSCII [answer]"))
			lines = append(lines, answerStyle.Width(w-2).Render(c.answerFormatted))
		} else {
			lines = append(lines, answerStyle.Width(w-2).Render("GOSCII [answer] "+c.mission.Answer))
		}
	}
	if c.state == stateFailed || c.state == stateIdle {
		lines = append(lines, keysStyle.Render(renderKeyBar(c)))
	}
	return strings.Join(lines, "\n")
}

func renderCockpitLines(c Cockpit, w int) []string {
	switch c.state {
	case stateRunning:
		return []string{"running..."}
	case stateAnalyzer:
		return []string{analyzerStyle.Render("GOSCII is reading the logs...")}
	case statePassed:
		lines := []string{passStyle.Render("PASSED")}
		if c.lastOutput != "" {
			lines = append(lines, outputStyle.Render(">> "+c.lastOutput))
		}
		return append(lines, keysStyle.Render("[ctrl+n] next   [ctrl+c] quit"))
	case stateFailed:
		lines := []string{failStyle.Width(w - 2).Render(c.lastOutput)}
		if c.analysis != "" {
			lines = append(lines, analyzerStyle.Width(w-2).Render("GOSCII ▸ "+c.analysis))
		}
		return lines
	}
	return nil
}

func renderKeyBar(c Cockpit) string {
	keys := "[ctrl+r] run   [ctrl+s] fmt"
	if c.mission.Difficulty != levels.Survival {
		keys += "   [ctrl+h] hint"
	}
	if c.signal != nil {
		keys += "   [ctrl+g] logs analyzer"
	}
	if c.mission.Answer != "" {
		keys += "   [ctrl+a] answer"
	}
	if c.state == stateFailed {
		if n := countErrorLines(c.lastOutput); n > 0 {
			errLabel := "[ctrl+e] goto error"
			if n > 1 {
				errLabel += fmt.Sprintf(" (%d)", n)
			}
			keys += "   " + errLabel
		}
	}
	return keys + "   [ctrl+c] quit"
}
