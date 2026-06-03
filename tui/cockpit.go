// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package tui implements the full terminal UI: home hub, signal setup, mission/difficulty popups, generator, and cockpit.
package tui

import (
	"context"
	"fmt"
	"slices"
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

// paneFocus selects which region receives scroll/typing input. alt+up/down cycles it.
type paneFocus int

const (
	focusEditor paneFocus = iota
	focusWorld
	focusStatus
)

type runDoneMsg engine.RunResult
type analyzerDoneMsg struct{ text string }
type analyzerErrMsg struct{ err error }

// Cockpit is the main mission screen.
type Cockpit struct {
	mission         *levels.Mission
	template        string
	header          string // scaffold shown above the editor (shown in full)
	footer          string // scaffold shown below the editor (closing braces, wired code)
	worldPort       viewport.Model
	statusPort      viewport.Model
	answerFormatted string // gofmt result of mission.Answer; empty for prose answers
	answerIsCode    bool   // true when answerFormatted is valid Go
	editor          textarea.Model
	signal          ai.Provider // nil in offline mode
	topic           string      // topic slug — selects the world art
	state           cockpitState
	focus           paneFocus
	lastOutput      string
	analysis        string
	hintIdx         int // -1 = hidden
	showAnswer      bool
	statusCollapsed bool
	worldCollapsed  bool // sounds dramatic, but it only hides the art/story block
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
	return Cockpit{
		mission:         ms,
		template:        tmpl,
		header:          engine.TemplateHeader(tmpl),
		footer:          engine.TemplateFooter(tmpl),
		worldPort:       viewport.New(),
		statusPort:      viewport.New(),
		answerFormatted: formatted,
		answerIsCode:    isCode,
		editor:          newEditor(),
		signal:          signal,
		topic:           topic,
		state:           stateIdle,
		focus:           focusEditor,
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
		c.relayout()
		return c, nil

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
		c.relayout()
		return c, nil

	case analyzerDoneMsg:
		c.state = stateFailed // back to failed so player can keep editing
		c.analysis = msg.text
		c.relayout()
		return c, nil

	case analyzerErrMsg:
		c.state = stateFailed
		c.analysis = "GOSCII signal lost. " + msg.err.Error()
		c.relayout()
		return c, nil
	}

	// Forward everything else (paste, cursor blink, focus, mouse) to the editor.
	var cmd tea.Cmd
	c.editor, cmd = c.editor.Update(msg)
	c.relayout()
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
			c.relayout()
			return c, c.runCode(), true
		}
	case "ctrl+h":
		if c.mission.Difficulty != levels.Survival {
			if n := len(c.mission.Hints); n > 0 {
				c.hintIdx = (c.hintIdx + 1) % n
			}
		}
		c.relayout()
		return c, nil, true
	case "ctrl+a":
		if c.mission.Answer != "" {
			c.showAnswer = !c.showAnswer
		}
		c.relayout()
		return c, nil, true
	case "ctrl+g":
		// GOSCII Logs Analyzer — only available on failure, only with a provider
		if c.state == stateFailed && c.signal != nil {
			c.state = stateAnalyzer
			c.analysis = ""
			c.relayout()
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
		c.focus = c.shiftFocus(-1)
		return c, nil, true
	case "alt+down":
		c.focus = c.shiftFocus(1)
		return c, nil, true
	case "up", "down", "pgup", "pgdown":
		if c.focus != focusEditor {
			c.scrollFocused(msg.String())
			return c, nil, true
		}
		return c.handleEditorKey(msg)
	case "ctrl+b":
		c.statusCollapsed = !c.statusCollapsed
		if c.statusCollapsed && c.focus == focusStatus {
			c.focus = focusEditor
		}
		c.relayout()
		return c, nil, true
	case "ctrl+t":
		c.worldCollapsed = !c.worldCollapsed
		if c.worldCollapsed && c.focus == focusWorld {
			c.focus = focusEditor
		}
		c.relayout()
		return c, nil, true
	default:
		if c.focus != focusEditor {
			return c, nil, true // a context pane is focused — swallow typing
		}
		return c.handleEditorKey(msg)
	}
	return c, nil, false
}

// --- focus ring (world ↕ editor ↕ status) ---

func (c Cockpit) focusRing() []paneFocus {
	ring := make([]paneFocus, 0, 3)
	if !c.worldCollapsed {
		ring = append(ring, focusWorld)
	}
	ring = append(ring, focusEditor)
	if !c.statusCollapsed {
		ring = append(ring, focusStatus)
	}
	return ring
}

func (c Cockpit) shiftFocus(delta int) paneFocus {
	ring := c.focusRing()
	idx := slices.Index(ring, c.focus)
	if idx < 0 {
		return focusEditor
	}
	n := len(ring)
	return ring[((idx+delta)%n+n)%n]
}

func (c *Cockpit) focusedPort() *viewport.Model {
	switch c.focus {
	case focusWorld:
		return &c.worldPort
	case focusStatus:
		return &c.statusPort
	}
	return nil
}

func (c *Cockpit) scrollFocused(key string) {
	vp := c.focusedPort()
	if vp == nil {
		return
	}
	switch key {
	case "up":
		vp.ScrollUp(1)
	case "down":
		vp.ScrollDown(1)
	case "pgup":
		vp.ScrollUp(vp.Height())
	case "pgdown":
		vp.ScrollDown(vp.Height())
	}
}

// --- layout ---

// relayout fits the regions to the terminal: world and status are scrollable
// panes capped at a third of the screen each; the header and footer scaffold
// are fixed and shown in full; the editor flexes to fill the rest (3-line floor).
func (c *Cockpit) relayout() {
	if c.width == 0 || c.height == 0 {
		return
	}
	paneW := c.width - 5 // inner width minus the one-column focus rail
	c.worldPort.SetWidth(paneW)
	c.worldPort.SetContent(worldContent(*c, paneW))
	c.statusPort.SetWidth(paneW)
	c.statusPort.SetContent(statusContent(*c, paneW))

	wH, sH := c.fitPanes()
	c.worldPort.SetHeight(wH)
	c.statusPort.SetHeight(sH)

	c.editor.SetWidth(c.width - 4)
	c.editor.SetHeight(max(3, c.height-c.rowsAboveEditor()-c.rowsBelowEditor()))
}

func (c Cockpit) fitPanes() (world, status int) {
	const editorMin = 3
	paneCap := max(6, c.height/3)

	fixed := 1 + 1 + 1 // separator + world toggle + status toggle
	if c.header != "" {
		fixed += lipgloss.Height(c.header)
	}
	if c.footer != "" {
		fixed += lipgloss.Height(c.footer)
	}

	wNat, sNat := 0, 0
	if !c.worldCollapsed {
		wNat = min(c.worldPort.TotalLineCount(), paneCap)
	}
	if !c.statusCollapsed {
		sNat = min(c.statusPort.TotalLineCount(), paneCap)
	}

	budget := max(0, c.height-fixed-editorMin)
	total := wNat + sNat
	if total == 0 || total <= budget {
		return wNat, sNat
	}
	world = budget * wNat / total
	status = budget - world
	return world, status
}

func (c Cockpit) rowsAboveEditor() int {
	rows := 1 // blank separator below the world region
	if c.worldCollapsed {
		rows++ // collapsed toggle line
	} else {
		rows += c.worldPort.Height() + 1 // content + toggle line
	}
	if c.header != "" {
		rows += lipgloss.Height(c.header)
	}
	return rows
}

func (c Cockpit) rowsBelowEditor() int {
	rows := 0
	if c.footer != "" {
		rows += lipgloss.Height(c.footer)
	}
	if c.statusCollapsed {
		rows++ // toggle + badge line
	} else {
		rows += 1 + c.statusPort.Height() // toggle line + content
	}
	return rows
}

func (c Cockpit) View() tea.View {
	c.relayout()

	var sb strings.Builder
	if c.worldCollapsed {
		sb.WriteString(styleHint.Render("[ctrl+t] ▶ "))
		sb.WriteString(headerStyle.Render(c.mission.Title))
	} else {
		sb.WriteString(paneBar(c.worldPort.View(), c.focus == focusWorld))
		sb.WriteString("\n")
		sb.WriteString(styleHint.Render("[ctrl+t] ▼"))
	}
	sb.WriteString("\n\n")
	if c.header != "" {
		sb.WriteString(templateHeaderStyle.Render(c.header))
		sb.WriteString("\n")
	}
	sb.WriteString(c.editor.View())
	sb.WriteString("\n")
	sb.WriteString(c.belowEditorView())

	v := tea.NewView(sb.String())
	// editor.Cursor() is relative to the editor; offset Y to absolute screen rows
	// (editor sits at column 0, so X is already correct). Show it only while the
	// editor holds focus.
	if c.focus == focusEditor {
		if cur := c.editor.Cursor(); cur != nil {
			cur.Y += c.rowsAboveEditor()
			v.Cursor = cur
		}
	}
	return v
}

// belowEditorView renders the footer, the [ctrl+b] toggle, and (when expanded)
// the scrollable status pane. Its height matches rowsBelowEditor.
func (c Cockpit) belowEditorView() string {
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
	sb.WriteString(paneBar(c.statusPort.View(), c.focus == focusStatus))
	return sb.String()
}

// paneBar draws a one-column rail down the right of a region, bright while the
// region holds focus.
func paneBar(s string, focused bool) string {
	rail := colorMuted
	if focused {
		rail = colorAccent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.Border{Right: "▌"}, false, true, false, false).
		BorderForeground(rail).
		Render(s)
}

var (
	templateHeaderStyle = lipgloss.NewStyle().Foreground(colorDim)
	analyzerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Italic(true)
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
	answerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	contractStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
)

// statusContent is the scrollable body of the status pane: the signal contract
// plus the live status lines (output, hints, answer, logs, key bar).
func statusContent(c Cockpit, w int) string {
	var lines []string
	if s := renderSignalContract(c); s != "" {
		lines = append(lines, s)
	}
	if s := renderStatus(c, w); s != "" {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}

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

func renderStatus(c Cockpit, w int) string {
	lines := renderCockpitLines(c, w)
	if hints := c.mission.Hints; c.hintIdx >= 0 && c.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", c.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w).Render("GOSCII ["+counter+"] "+hints[c.hintIdx]))
	}
	if c.showAnswer {
		if c.answerIsCode {
			lines = append(lines, answerStyle.Render("GOSCII [answer]"))
			lines = append(lines, answerStyle.Width(w).Render(c.answerFormatted))
		} else {
			lines = append(lines, answerStyle.Width(w).Render("GOSCII [answer] "+c.mission.Answer))
		}
	}
	if c.state == stateFailed || c.state == stateIdle {
		lines = append(lines, styleHint.Width(w).Render(renderKeyBar(c)))
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
		return append(lines, styleHint.Width(w).Render("[ctrl+n] next   [esc] hub   [ctrl+c] quit"))
	case stateFailed:
		lines := []string{styleFail.Width(w).Render(c.lastOutput)}
		if c.analysis != "" {
			lines = append(lines, analyzerStyle.Width(w).Render("GOSCII ▸ "+c.analysis))
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
	return keys + "   [alt ↑↓] panes   [esc] hub   [ctrl+c] quit"
}
