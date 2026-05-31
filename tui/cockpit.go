// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package tui implements the full terminal UI: home hub, signal setup, mission/difficulty popups, generator, and cockpit.
package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	footer          string // read-only template lines shown below the editor (closing braces, wired code)
	answerFormatted string // gofmt result of mission.Answer; empty for prose answers
	answerIsCode    bool   // true when answerFormatted is valid Go
	editor          textarea.Model
	signal          ai.Provider // nil in offline mode
	topic           string      // topic slug — selects the world art
	state           cockpitState
	lastOutput      string
	analysis        string
	hintIdx         int // -1 = hidden
	showAnswer      bool
	statusCollapsed bool
	worldCollapsed  bool // not a drama: hides the world block (title/art/story)
	step            int  // current mission in chain (0 = no chain)
	maxStep         int  // total missions in chain
	width           int
	height          int
}

// Passed reports whether the player completed the level successfully.
func (c Cockpit) Passed() bool { return c.state == statePassed }

// New creates a cockpit model. signal may be nil (offline mode).
func New(ms *levels.Mission, tmpl string, signal ai.Provider, step, maxStep int, topic string) Cockpit {
	formatted, isCode := formatGoSnippet(ms.Answer)
	hdr := engine.TemplateHeader(tmpl)
	hdrPort := viewport.New()
	hdrPort.SetContent(hdr)
	return Cockpit{
		mission:         ms,
		template:        tmpl,
		header:          hdr,
		hdrPort:         hdrPort,
		footer:          engine.TemplateFooter(tmpl),
		answerFormatted: formatted,
		answerIsCode:    isCode,
		editor:          newEditor(),
		signal:          signal,
		topic:           topic,
		state:           stateIdle,
		hintIdx:         -1,
		step:            step,
		maxStep:         maxStep,
	}
}

func (c Cockpit) PlayerCode() string {
	if c.state == statePassed {
		return c.editor.Value()
	}
	return ""
}

func (c Cockpit) Init() tea.Cmd {
	return nil
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

func (c Cockpit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m, cmd, handled := c.handleKey(msg); handled {
			return m, cmd
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.editor.SetWidth(msg.Width - 4)
		if c.header != "" {
			const maxHeaderLines = 6
			hdrH := min(strings.Count(c.header, "\n")+1, maxHeaderLines)
			c.hdrPort.SetWidth(msg.Width - 4)
			c.hdrPort.SetHeight(hdrH)
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

func (c Cockpit) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) { //nolint:gocyclo // flat key dispatch
	switch msg.String() {
	case "ctrl+c":
		return c, tea.Quit, true
	case "esc":
		return c, func() tea.Msg { return BackMsg{} }, true
	case "ctrl+r":
		if c.state != stateRunning && c.state != stateAnalyzer {
			c.state = stateRunning
			c.analysis = ""
			return c, c.runCode(), true
		}
	case "ctrl+h":
		if c.mission.Difficulty != levels.Survival {
			if n := len(c.mission.Hints); n > 0 {
				c.hintIdx = (c.hintIdx + 1) % n
			}
		}
		return c, nil, true
	case "ctrl+a":
		if c.mission.Answer != "" {
			c.showAnswer = !c.showAnswer
		}
		return c, nil, true
	case "ctrl+g":
		// GOSCII Logs Analyzer — only available on failure, only with a provider
		if c.state == stateFailed && c.signal != nil {
			c.state = stateAnalyzer
			c.analysis = ""
			return c, c.runAnalyzer(), true
		}
	case "ctrl+n":
		if c.state == statePassed {
			return c, func() tea.Msg { return NextMsg{} }, true
		}
	case "ctrl+e":
		if c.state == stateFailed {
			if line, col := parseFirstError(c.lastOutput); line > 0 {
				for c.editor.Line() < line-1 {
					c.editor.CursorDown()
				}
				for c.editor.Line() > line-1 {
					c.editor.CursorUp()
				}
				if col > 0 {
					c.editor.SetCursorColumn(col - 1)
				}
			}
		}
		return c, nil, true
	case "alt+up":
		c.hdrPort.ScrollUp(1)
		return c, nil, true
	case "alt+down":
		c.hdrPort.ScrollDown(1)
		return c, nil, true
	case "ctrl+b":
		c.statusCollapsed = !c.statusCollapsed
		c.recalcEditorHeight()
		return c, nil, true
	case "ctrl+t":
		c.worldCollapsed = !c.worldCollapsed
		c.recalcEditorHeight()
		return c, nil, true
	default:
		return c.handleEditorKey(msg)
	}
	return c, nil, false
}

func (c Cockpit) View() tea.View {
	c.recalcEditorHeight() // size the editor to the current layout every frame

	var sb strings.Builder
	sb.WriteString(renderWorld(c))
	sb.WriteString("\n\n")
	if c.hdrPort.Height() > 0 {
		sb.WriteString(templateHeaderStyle.Render(c.hdrPort.View()))
		sb.WriteString("\n")
	}
	sb.WriteString(c.editor.View())
	sb.WriteString("\n")
	sb.WriteString(belowEditor(c))

	v := tea.NewView(sb.String())
	// editor.Cursor() is relative to the editor; offset Y to absolute screen rows
	// (editor sits at column 0, so X is already correct).
	if cur := c.editor.Cursor(); cur != nil {
		cur.Y += editorTopRow(c)
		v.Cursor = cur
	}
	return v
}

// belowEditor renders everything under the editor: footer, the [ctrl+b] toggle,
// and (when expanded) the signal contract and status panel. Its height drives
// recalcEditorHeight.
func belowEditor(c Cockpit) string {
	var sb strings.Builder
	if c.footer != "" {
		sb.WriteString(templateHeaderStyle.Render(c.footer))
		sb.WriteString("\n")
	}
	if c.statusCollapsed {
		sb.WriteString(styleHint.Render("[ctrl+b] ▶"))
		switch c.state {
		case statePassed:
			sb.WriteString(" ")
			sb.WriteString(stylePass.Render("PASSED"))
		case stateFailed:
			sb.WriteString(" ")
			sb.WriteString(styleFail.Render("FAILED"))
		case stateRunning:
			sb.WriteString(" running...")
		case stateAnalyzer:
			sb.WriteString(" ")
			sb.WriteString(analyzerStyle.Render("logs analyzer..."))
		}
		return sb.String()
	}
	sb.WriteString(styleHint.Render("[ctrl+b] ▼"))
	sb.WriteString("\n")
	if signal := renderSignalContract(c); signal != "" {
		sb.WriteString(signal)
		sb.WriteString("\n")
	}
	sb.WriteString(renderStatus(c))
	return sb.String()
}

var (
	templateHeaderStyle = lipgloss.NewStyle().Foreground(colorDim)
	analyzerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Italic(true)
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	contractStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
)

// renderSignalContract shows the expected output to the player on all difficulties except survival.
func renderSignalContract(c Cockpit) string {
	if c.mission.Difficulty == levels.Survival {
		return ""
	}
	check := c.mission.Check
	switch {
	case check.StdoutEquals != "":
		return contractStyle.Render("SIGNAL ▸ target output → ") + styleAccent.Render(check.StdoutEquals)
	case check.StdoutContains != "":
		return contractStyle.Render("SIGNAL ▸ output must contain → ") + styleAccent.Render(check.StdoutContains)
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
		lines = append(lines, styleHint.Width(w-2).Render(renderKeyBar(c)))
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
		lines := []string{stylePass.Render("PASSED")}
		if c.lastOutput != "" {
			lines = append(lines, outputStyle.Render(">> "+c.lastOutput))
		}
		return append(lines, styleHint.Width(w-2).Render("[ctrl+n] next   [esc] hub   [ctrl+c] quit"))
	case stateFailed:
		lines := []string{styleFail.Width(w - 2).Render(c.lastOutput)}
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
		keys += "   [ctrl+g] log analyzer"
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
	return keys + "   [esc] hub   [ctrl+c] quit"
}
