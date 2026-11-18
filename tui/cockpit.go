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
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
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
	stateAnalyzer // GOSCII is reading the logs or answering
	stateDepleted // survival oxygen ran out
)

// paneFocus selects which region receives scroll/typing input. alt+up/down cycles it.
type paneFocus int

const (
	focusEditor paneFocus = iota
	focusWorld
	focusStatus
)

// runDoneMsg and partnerMsg carry the id of the cockpit's work, so a reply that
// lands after the player moved on is dropped instead of judging another mission.
type runDoneMsg struct {
	id      uint64
	verdict engine.Verdict
}

type partnerMsg struct {
	id      uint64
	text    string
	err     error
	debrief bool
}

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
	comms           textinput.Model // question line to GOSCII
	asking          bool
	signal          ai.Provider      // nil in offline mode
	progress        *engine.Progress // sets GOSCII's voice; may be nil
	oxygen          *int             // survival O2 in runs, shared across the chain; nil = no limit
	final           bool             // last mission of a chain: no hints
	topic           string           // topic slug — selects the world art
	state           cockpitState
	before          cockpitState // state to return to after GOSCII answers
	focus           paneFocus
	lastOutput      string
	analysis        string
	analyses        []string // earlier partner replies on this mission
	debrief         string
	runs            int
	hintIdx         int // -1 = hidden
	showAnswer      bool
	statusCollapsed bool
	worldCollapsed  bool // sounds dramatic, but it only hides the art/story block
	step            int  // current mission in chain (0 = no chain)
	maxStep         int  // total missions in chain
	work            *work
	width           int
	height          int
}

// Passed reports whether the player completed the level successfully.
func (c Cockpit) Passed() bool { return c.state == statePassed }

func (c Cockpit) decode() bool { return c.mission.Check.Prediction }

// hintsOff reports whether hints are withheld: survival and chain finales.
func (c Cockpit) hintsOff() bool { return c.mission.Difficulty == levels.Survival || c.final }

// New creates a cockpit model. signal may be nil (offline mode).
func New(ms *levels.Mission, tmpl string, signal ai.Provider, step, maxStep int, topic string) Cockpit {
	formatted, isCode := formatGoSnippet(ms.Answer)
	c := Cockpit{
		mission:         ms,
		template:        tmpl,
		header:          engine.DisplayScaffold(engine.TemplateHeader(tmpl)),
		footer:          engine.TemplateFooter(tmpl),
		worldPort:       viewport.New(),
		statusPort:      viewport.New(),
		answerFormatted: formatted,
		answerIsCode:    isCode,
		editor:          newEditor(),
		comms:           newComms(),
		signal:          signal,
		topic:           topic,
		state:           stateIdle,
		focus:           focusEditor,
		hintIdx:         -1,
		step:            step,
		maxStep:         maxStep,
		work:            newWork(),
	}
	if c.decode() {
		// the program is read, not edited: it goes in the scrollable world pane
		c.header = ""
		c.footer = ""
		c.answerIsCode = false
		c.editor.Placeholder = "predicted output, line by line"
	}
	if ms.Seed != "" {
		seed, ok := formatGoSnippet(ms.Seed)
		if !ok {
			seed = strings.ReplaceAll(ms.Seed, "\t", "    ")
		}
		c.editor.SetValue(seed)
	}
	return c
}

func newComms() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "ask GOSCII (enter on empty reads the last fault log)"
	ti.Prompt = "GOSCII ◂ "
	return ti
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
	check := c.mission.Check
	id := c.work.id
	return func() tea.Msg {
		return runDoneMsg{id: id, verdict: engine.Judge(tmpl, code, check)}
	}
}

// stop cancels GOSCII requests still in flight for this mission.
func (c Cockpit) stop() { c.work.stop() }

func (c Cockpit) repaired() int { return c.progress.Repaired() }

func (c Cockpit) coach() ai.Coach {
	expected := ""
	if !c.decode() && c.mission.Difficulty != levels.Survival {
		expected = c.mission.Check.StdoutEquals
	}
	return ai.Coach{
		Concept:  c.mission.Concept,
		Beat:     c.mission.Beat,
		Kind:     string(c.mission.Kind),
		Story:    c.mission.Story,
		Code:     c.editor.Value(),
		Expected: expected,
		Got:      c.lastOutput,
		Attempt:  c.runs,
		Hints:    c.hintIdx + 1,
		Previous: c.analyses,
		Repaired: c.repaired(),
	}
}

func partnerCmd(w *work, f func(ctx context.Context) (string, ai.Usage, error), debrief bool) tea.Cmd {
	ctx, cancel := w.context(2 * time.Minute)
	id := w.id
	return func() tea.Msg {
		defer cancel()
		text, _, err := f(ctx)
		return partnerMsg{id: id, text: text, err: err, debrief: debrief}
	}
}

func (c Cockpit) runAnalyzer(question string) tea.Cmd {
	if c.signal == nil {
		return nil
	}
	sig, coach := c.signal, c.coach()
	if question == "" {
		return partnerCmd(c.work, func(ctx context.Context) (string, ai.Usage, error) { return ai.Analyze(ctx, sig, coach) }, false)
	}
	return partnerCmd(c.work, func(ctx context.Context) (string, ai.Usage, error) { return ai.Ask(ctx, sig, coach, question) }, false)
}

func (c Cockpit) runDebrief() tea.Cmd {
	if c.signal == nil || c.decode() || c.mission.Answer == "" {
		return nil
	}
	sig, coach, ref := c.signal, c.coach(), c.mission.Answer
	return partnerCmd(c.work, func(ctx context.Context) (string, ai.Usage, error) { return ai.Debrief(ctx, sig, coach, ref) }, true)
}

func (c Cockpit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if c.asking {
			return c.handleComms(msg)
		}
		if m, cmd, handled := c.handleKey(msg); handled {
			return m, cmd
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.relayout()
		return c, nil

	case runDoneMsg:
		if msg.id != c.work.id {
			return c, nil
		}
		return c.judged(msg.verdict)

	case partnerMsg:
		if msg.id != c.work.id {
			return c, nil
		}
		if msg.debrief {
			if msg.err == nil {
				c.debrief = msg.text
			}
			c.relayout()
			return c, nil
		}
		c.state = c.before
		if msg.err != nil {
			c.analysis = "GOSCII signal lost. " + msg.err.Error()
		} else {
			c.analysis = msg.text
			c.analyses = append(c.analyses, msg.text)
		}
		c.relayout()
		return c, nil
	}

	if c.asking {
		var cmd tea.Cmd
		c.comms, cmd = c.comms.Update(msg)
		return c, cmd
	}
	// Forward everything else (paste, cursor blink, focus, mouse) to the editor.
	var cmd tea.Cmd
	c.editor, cmd = c.editor.Update(msg)
	c.relayout()
	return c, cmd
}

func (c Cockpit) judged(v engine.Verdict) (tea.Model, tea.Cmd) {
	res := v.Result
	if v.Passed {
		c.state = statePassed
		c.lastOutput = strings.TrimSpace(res.Stdout)
		c.relayout()
		return c, c.runDebrief()
	}
	c.state = stateFailed
	switch {
	case res.Stderr != "":
		c.lastOutput = res.Stderr
	case v.Failure != "":
		c.lastOutput = v.Failure
	default:
		c.lastOutput = fmt.Sprintf("got: %q", strings.TrimSpace(res.Stdout))
	}
	if c.oxygen != nil && *c.oxygen <= 0 {
		c.state = stateDepleted
	}
	c.relayout()
	return c, nil
}

func (c Cockpit) handleComms(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		c.asking = false
		c.comms.Blur()
		c.relayout()
		return c, c.editor.Focus()
	case "enter":
		q := strings.TrimSpace(c.comms.Value())
		if q == "" && c.state != stateFailed {
			return c, nil
		}
		c.asking = false
		c.comms.SetValue("")
		c.comms.Blur()
		c.before = c.state
		c.state = stateAnalyzer
		c.analysis = ""
		c.relayout()
		return c, tea.Batch(c.editor.Focus(), c.runAnalyzer(q))
	case "ctrl+c":
		return c, tea.Quit
	}
	var cmd tea.Cmd
	c.comms, cmd = c.comms.Update(msg)
	return c, cmd
}

func (c Cockpit) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) { //nolint:gocyclo // flat key dispatch
	switch msg.String() {
	case "ctrl+c":
		return c, tea.Quit, true
	case "esc":
		return c, func() tea.Msg { return BackMsg{} }, true
	case "ctrl+r":
		if c.state == stateRunning || c.state == stateAnalyzer || c.state == stateDepleted {
			return c, nil, true
		}
		if c.oxygen != nil {
			if *c.oxygen <= 0 {
				c.state = stateDepleted
				c.relayout()
				return c, nil, true
			}
			*c.oxygen--
		}
		c.runs++
		c.state = stateRunning
		c.analysis = ""
		c.relayout()
		return c, c.runCode(), true
	case "ctrl+h":
		if !c.hintsOff() {
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
		if c.signal != nil && c.state != stateRunning && c.state != stateAnalyzer {
			c.asking = true
			c.editor.Blur()
			c.relayout()
			return c, c.comms.Focus(), true
		}
		return c, nil, true
	case "ctrl+n":
		if c.state == statePassed {
			return c, func() tea.Msg { return NextMsg{} }, true
		}
	case "ctrl+s":
		if c.decode() {
			return c, nil, true
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
	}
	if c.focus != focusEditor {
		return c, nil, true // a context pane is focused — swallow typing
	}
	if c.decode() {
		return c, nil, false // predictions are plain text: no auto-pairs or gofmt
	}
	return c.handleEditorKey(msg)
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
	c.comms.SetWidth(paneW - 10)

	wH, sH := c.fitPanes()
	c.worldPort.SetHeight(wH)
	c.statusPort.SetHeight(sH)

	c.editor.SetWidth(c.width - 4)
	c.editor.SetHeight(max(3, c.height-c.rowsAboveEditor()-c.rowsBelowEditor()))
}

func (c Cockpit) fitPanes() (world, status int) {
	const editorMin = 3
	paneCap := max(6, c.height/3)
	if c.decode() {
		paneCap = max(6, c.height*2/3) // the program needs the room more than the short prediction
	}

	fixed := 1 + 1 + 1 // separator + world toggle + status toggle
	if c.header != "" {
		fixed += lipgloss.Height(c.header)
	}
	if c.footer != "" {
		fixed += lipgloss.Height(c.footer)
	}
	if c.asking {
		fixed++
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
	if c.asking {
		rows++
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
	if c.focus == focusEditor && !c.asking {
		if cur := c.editor.Cursor(); cur != nil {
			cur.Y += c.rowsAboveEditor()
			v.Cursor = cur
		}
	}
	return v
}

// belowEditorView renders the footer, the comms line, the [ctrl+b] toggle, and
// (when expanded) the scrollable status pane. Its height matches rowsBelowEditor.
func (c Cockpit) belowEditorView() string {
	var sb strings.Builder
	if c.footer != "" {
		sb.WriteString(templateHeaderStyle.Render(c.footer))
		sb.WriteString("\n")
	}
	if c.asking {
		sb.WriteString(c.comms.View())
		sb.WriteString("\n")
	}
	if c.statusCollapsed {
		sb.WriteString(styleHint.Render("[ctrl+b] ▶"))
		switch c.state {
		case statePassed:
			sb.WriteString(" " + stylePass.Render("PASSED"))
		case stateFailed:
			sb.WriteString(" " + styleFail.Render("FAILED"))
		case stateDepleted:
			sb.WriteString(" " + styleFail.Render("O2 DEPLETED"))
		case stateRunning:
			sb.WriteString(" running...")
		case stateAnalyzer:
			sb.WriteString(" " + analyzerStyle.Render("GOSCII is thinking..."))
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

// renderSignalContract shows what GOSCII will accept: the target output on all
// difficulties except survival, plus construct rules and shifted readings.
func renderSignalContract(c Cockpit) string {
	check := c.mission.Check
	var lines []string
	if c.oxygen != nil {
		lines = append(lines, contractStyle.Render("LIFE SUPPORT ▸ ")+oxygenGauge(*c.oxygen))
	}
	if check.Prediction {
		lines = append(lines, contractStyle.Render("SIGNAL ▸ predict the exact output of the recovered program"))
		return strings.Join(lines, "\n")
	}
	if c.mission.Difficulty != levels.Survival {
		switch {
		case check.StdoutEquals != "":
			lines = append(lines, contractStyle.Render("SIGNAL ▸ target output → ")+styleAccent.Render(check.StdoutEquals))
		case check.StdoutContains != "":
			lines = append(lines, contractStyle.Render("SIGNAL ▸ output must contain → ")+styleAccent.Render(check.StdoutContains))
		case check.StdoutNonempty:
			lines = append(lines, contractStyle.Render("SIGNAL ▸ any output accepted"))
		}
	}
	if len(check.Requires) > 0 {
		lines = append(lines, contractStyle.Render("PROTOCOL ▸ must use → ")+styleAccent.Render(describeRules(check.Requires)))
	}
	if len(check.Forbids) > 0 {
		lines = append(lines, contractStyle.Render("PROTOCOL ▸ must not use → ")+styleAccent.Render(describeRules(check.Forbids)))
	}
	if n := len(check.Cases); n > 0 {
		lines = append(lines, contractStyle.Render(fmt.Sprintf("SIGNAL ▸ +%d shifted readings checked after the sample", n)))
	}
	return strings.Join(lines, "\n")
}

func describeRules(rules []string) string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = engine.DescribeRule(r)
	}
	return strings.Join(out, ", ")
}

func oxygenGauge(o2 int) string {
	style := stylePass
	if o2 <= 3 {
		style = styleFail
	}
	return style.Render(fmt.Sprintf("O2 %d runs", max(0, o2)))
}

func renderStatus(c Cockpit, w int) string {
	lines := renderCockpitLines(c, w)
	if hints := c.mission.Hints; c.hintIdx >= 0 && c.hintIdx < len(hints) {
		counter := fmt.Sprintf("%d/%d", c.hintIdx+1, len(hints))
		lines = append(lines, hintStyle.Width(w).Render("GOSCII ["+counter+"] "+hints[c.hintIdx]))
	}
	if c.showAnswer {
		switch {
		case c.answerIsCode:
			lines = append(lines, answerStyle.Render("GOSCII [answer]"), answerStyle.Width(w).Render(c.answerFormatted))
		case c.decode():
			lines = append(lines, answerStyle.Render("GOSCII [recovered log]"), answerStyle.Width(w).Render(c.mission.Answer))
		default:
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
		if c.debrief != "" {
			lines = append(lines, analyzerStyle.Width(w).Render("GOSCII ▸ "+c.debrief))
		}
		return append(lines, styleHint.Width(w).Render("[ctrl+n] next   [esc] hub   [ctrl+c] quit"))
	case stateDepleted:
		return []string{
			styleFail.Width(w).Render("O2 DEPLETED. Life support is down; this chain is lost."),
			styleFail.Width(w).Render(c.lastOutput),
			styleHint.Width(w).Render("[esc] hub   [ctrl+c] quit"),
		}
	case stateFailed:
		lines := []string{styleFail.Width(w).Render(c.lastOutput)}
		if c.analysis != "" {
			lines = append(lines, analyzerStyle.Width(w).Render("GOSCII ▸ "+c.analysis))
		}
		return lines
	case stateIdle:
		if c.analysis != "" {
			return []string{analyzerStyle.Width(w).Render("GOSCII ▸ " + c.analysis)}
		}
	}
	return nil
}

func renderKeyBar(c Cockpit) string {
	keys := "[ctrl+r] run"
	if c.decode() {
		keys = "[ctrl+r] transmit prediction"
	} else {
		keys += "   [ctrl+s] fmt"
	}
	if !c.hintsOff() {
		keys += "   [ctrl+h] hint"
	}
	if c.signal != nil {
		keys += "   [ctrl+g] ask GOSCII"
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
