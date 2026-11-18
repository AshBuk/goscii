// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Generator drives a mission chain: it forges each mission through the signal,
// shows the pipeline stages while it waits, forges the next mission in the
// background, and hands each verified mission to the cockpit.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

type genState int

const (
	genForging genState = iota
	genCockpit
	genFailed
	genComplete
)

// oxygenPerMission is the survival O2 budget, in runs, per mission in the chain.
const oxygenPerMission = 4

type forgeEventMsg struct {
	gen int
	ev  forge.Event
	ch  <-chan any
}

type forgeDoneMsg struct {
	gen      int
	res      *forge.Result
	err      error
	prefetch bool
}

type logRecoveredMsg struct{ entry engine.LogEntry }

// jobs holds the cancel funcs of in-flight forges; shared across model copies.
type jobs struct{ cancels []context.CancelFunc }

func (j *jobs) add(c context.CancelFunc) { j.cancels = append(j.cancels, c) }

func (j *jobs) stop() {
	for _, c := range j.cancels {
		c()
	}
	j.cancels = nil
}

// GeneratorModel forges a chain of missions and owns the cockpit once one is ready.
type GeneratorModel struct {
	state       genState
	signal      ai.Provider
	chain       *forge.Chain
	progress    *engine.Progress
	prefetchOn  bool
	gen         int
	ev          forge.Event
	lastErr     error
	mission     *levels.Mission
	template    string
	cockpit     *Cockpit
	next        *forgeDoneMsg // prefetched mission waiting for its turn
	prefetching bool
	awaiting    bool // the player is waiting on the prefetch
	oxygen      *int
	usage       ai.Usage
	repairs     []string // station systems repaired in this chain
	recovered   *engine.LogEntry
	jobs        *jobs
	spin        spinner.Model
	width       int
	height      int
}

// NewGeneratorModel starts a chain for one topic, difficulty and kind.
func NewGeneratorModel(signal ai.Provider, chain *forge.Chain, cfg engine.Config, p *engine.Progress) GeneratorModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)
	g := GeneratorModel{
		signal:     signal,
		chain:      chain,
		progress:   p,
		prefetchOn: cfg.PrefetchOn(),
		jobs:       &jobs{},
		spin:       sp,
	}
	if chain.Difficulty == levels.Survival {
		o2 := oxygenPerMission * chain.Total()
		g.oxygen = &o2
	}
	return g
}

func (g GeneratorModel) Init() tea.Cmd {
	return tea.Batch(g.startForge(), g.spin.Tick)
}

func (g *GeneratorModel) startForge() tea.Cmd {
	g.gen++
	g.state = genForging
	g.ev = forge.Event{Draft: 1}
	g.lastErr = nil
	return forgeCmd(g.jobs, g.signal, g.chain.Next(g.progress), g.gen, false)
}

// forgeCmd runs the forge in a goroutine and streams its events back.
func forgeCmd(j *jobs, sig ai.Provider, o forge.Order, gen int, prefetch bool) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	j.add(cancel)
	ch := make(chan any, 32)
	go func() {
		defer cancel()
		res, err := forge.Forge(ctx, sig, o, func(e forge.Event) {
			select {
			case ch <- e:
			default: // a slow screen never stalls the forge
			}
		})
		ch <- forgeDoneMsg{gen: gen, res: res, err: err, prefetch: prefetch}
		close(ch)
	}()
	if prefetch {
		return waitDone(ch)
	}
	return waitForge(ch, gen)
}

func waitForge(ch <-chan any, gen int) tea.Cmd {
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			return nil
		}
		if e, isEvent := v.(forge.Event); isEvent {
			return forgeEventMsg{gen: gen, ev: e, ch: ch}
		}
		return v
	}
}

// waitDone drains events silently and returns only the result.
func waitDone(ch <-chan any) tea.Cmd {
	return func() tea.Msg {
		for v := range ch {
			if done, ok := v.(forgeDoneMsg); ok {
				return done
			}
		}
		return nil
	}
}

func (g GeneratorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocyclo // bubbletea "one switch" updates
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width, g.height = msg.Width, msg.Height
		if g.cockpit != nil {
			return g.forward(msg)
		}
		return g, nil

	case BackMsg:
		g.jobs.stop()
		return g, func() tea.Msg { return BackMsg{} }

	case spinner.TickMsg:
		if g.state == genForging {
			var cmd tea.Cmd
			g.spin, cmd = g.spin.Update(msg)
			return g, cmd
		}
		return g, nil

	case forgeEventMsg:
		if msg.gen != g.gen || g.state != genForging {
			return g, waitForge(msg.ch, msg.gen)
		}
		g.ev = msg.ev
		return g, waitForge(msg.ch, msg.gen)

	case forgeDoneMsg:
		return g.forged(msg)

	case logRecoveredMsg:
		g.recovered = &msg.entry
		g.progress.Logs = append(g.progress.Logs, msg.entry)
		return g, saveProgressCmd(g.progress)

	case runDoneMsg:
		cmd := g.maybePrefetch()
		m, fwd := g.forward(msg)
		return m, tea.Batch(cmd, fwd)

	case NextMsg:
		return g.advance()

	case tea.KeyPressMsg:
		if m, cmd, ok := g.handleKey(msg); ok {
			return m, cmd
		}
	}

	if g.state == genCockpit && g.cockpit != nil {
		return g.forward(msg)
	}
	return g, nil
}

func (g GeneratorModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() == "ctrl+c" && g.state != genCockpit {
		return g, tea.Quit, true
	}
	switch g.state {
	case genFailed:
		switch msg.String() {
		case "r", "R":
			return g, tea.Batch(g.startForge(), g.spin.Tick), true
		case "m", "M":
			g.jobs.stop()
			return g, func() tea.Msg { return NewMissionMsg{} }, true
		case "h", "H", "esc":
			g.jobs.stop()
			return g, func() tea.Msg { return BackMsg{} }, true
		}
		return g, nil, true
	case genForging:
		if msg.String() == "esc" {
			g.jobs.stop()
			return g, func() tea.Msg { return BackMsg{} }, true
		}
		return g, nil, true
	case genComplete:
		return g, func() tea.Msg { return BackMsg{} }, true
	}
	return g, nil, false
}

func (g GeneratorModel) forward(msg tea.Msg) (tea.Model, tea.Cmd) {
	newC, cmd := g.cockpit.Update(msg)
	c := newC.(Cockpit)
	g.cockpit = &c
	return g, cmd
}

func (g GeneratorModel) forged(msg forgeDoneMsg) (tea.Model, tea.Cmd) {
	if msg.prefetch {
		g.prefetching = false
		if msg.err != nil {
			if g.awaiting { // the player is waiting: forge in the foreground instead
				g.awaiting = false
				return g, tea.Batch(g.startForge(), g.spin.Tick)
			}
			return g, nil
		}
		if g.awaiting {
			g.awaiting = false
			return g.launch(msg.res)
		}
		g.next = &msg
		return g, nil
	}
	if msg.gen != g.gen {
		return g, nil
	}
	if msg.err != nil {
		g.state = genFailed
		g.lastErr = msg.err
		return g, nil
	}
	return g.launch(msg.res)
}

func (g GeneratorModel) launch(res *forge.Result) (tea.Model, tea.Cmd) {
	g.usage.Add(res.Usage)
	c := New(res.Mission, res.Template, g.signal, g.chain.Step(), g.chain.Total(), res.Mission.Concept)
	c.progress = g.progress
	c.oxygen = g.oxygen
	c.final = g.chain.Final()
	sized, _ := c.Update(tea.WindowSizeMsg{Width: g.width, Height: g.height})
	c = sized.(Cockpit)
	g.mission = res.Mission
	g.template = res.Template
	g.cockpit = &c
	g.state = genCockpit
	rec := forge.Record{Mission: res.Mission, Template: res.Template, Signal: g.signal.Name(), Model: g.signal.Model(), ForgedAt: time.Now()}
	return g, tea.Batch(c.Init(), func() tea.Msg { _ = forge.Archive(rec); return nil })
}

// maybePrefetch starts forging the next mission after the player's first run,
// so tokens are not spent on a mission abandoned at once.
func (g *GeneratorModel) maybePrefetch() tea.Cmd {
	if !g.prefetchOn || g.prefetching || g.next != nil || g.state != genCockpit || g.mission == nil {
		return nil
	}
	o, ok := g.chain.Peek(g.progress, g.entry(g.mission.Answer))
	if !ok {
		return nil
	}
	g.prefetching = true
	return forgeCmd(g.jobs, g.signal, o, g.gen, true)
}

func (g GeneratorModel) entry(code string) forge.Entry {
	return forge.Entry{
		Title:      g.mission.Title,
		Story:      g.mission.Story,
		Beat:       g.mission.Beat,
		System:     g.mission.System,
		PlayerCode: code,
		Program:    engine.InjectCode(g.template, code),
	}
}

func (g GeneratorModel) advance() (tea.Model, tea.Cmd) {
	if g.cockpit == nil || !g.cockpit.Passed() {
		return g, nil
	}
	code := g.cockpit.PlayerCode()
	g.chain.Record(g.entry(code))
	g.progress.RecordCompletion(g.chain.Topic.Slug, engine.CompletedRun{
		MissionID:  g.mission.ID,
		Difficulty: string(g.mission.Difficulty),
		Beat:       g.mission.Beat,
		Kind:       string(g.mission.Kind),
		Runs:       g.cockpit.runs,
	})
	if g.mission.System != "" {
		g.progress.Station.Repair(g.mission.System, repairGain(g.mission.Difficulty))
		g.repairs = append(g.repairs, g.mission.System)
	}
	id := g.mission.ID
	cmds := []tea.Cmd{saveProgressCmd(g.progress), func() tea.Msg { forge.MarkSolved(id); return nil }}

	if g.chain.Done() {
		g.jobs.stop()
		g.state = genComplete
		g.cockpit = nil
		return g, tea.Batch(append(cmds, recoverLogCmd(g.signal, g.progress))...)
	}
	g.cockpit = nil
	if g.next != nil {
		res := g.next.res
		g.next = nil
		m, cmd := g.launch(res)
		return m, tea.Batch(append(cmds, cmd)...)
	}
	if g.prefetching {
		g.state = genForging
		g.awaiting = true
		g.ev = forge.Event{Draft: 1}
		return g, tea.Batch(append(cmds, g.spin.Tick)...)
	}
	return g, tea.Batch(append(cmds, g.startForge(), g.spin.Tick)...)
}

func repairGain(d levels.Difficulty) int {
	switch d {
	case levels.Medium:
		return 10
	case levels.Hard:
		return 14
	case levels.Survival:
		return 20
	default:
		return 6
	}
}

// recoverLogCmd unlocks the next crew log fragment and lets GOSCII voice it.
func recoverLogCmd(sig ai.Provider, p *engine.Progress) tea.Cmd {
	facts := levels.CrewLogs()
	idx := len(p.Logs)
	if idx >= len(facts) {
		return nil
	}
	fact := facts[idx]
	repaired := p.Repaired()
	return func() tea.Msg {
		entry := engine.LogEntry{ID: fact.ID, Text: fact.Fact}
		if sig != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if text, _, err := ai.RecoverLog(ctx, sig, fact.Fact, repaired); err == nil && text != "" {
				entry.Text = text
			}
		}
		return logRecoveredMsg{entry: entry}
	}
}

func (g GeneratorModel) View() tea.View {
	switch g.state {
	case genCockpit:
		if g.cockpit != nil {
			return g.cockpit.View()
		}
	case genFailed:
		return tea.NewView(g.viewFailed())
	case genComplete:
		return tea.NewView(g.viewComplete())
	}
	return tea.NewView(g.viewForging())
}

func (g GeneratorModel) viewForging() string {
	cw := contentWidth(g.width)
	label := fmt.Sprintf("Establishing %s signal...", g.chain.Difficulty)
	if total := g.chain.Total(); total > 1 {
		label += fmt.Sprintf("  [%d/%d]", g.chain.Step(), total)
	}
	stage := g.ev.Stage.String()
	if g.awaiting {
		stage = "receiving the next transmission"
	}
	if g.ev.Draft > 1 {
		stage += fmt.Sprintf(" · draft %d", g.ev.Draft)
	}
	if g.ev.Repair > 0 {
		stage += fmt.Sprintf(" · recalibration %d", g.ev.Repair)
	}
	status := g.spin.View() + " " + styleMuted.Render(fmt.Sprintf("%q · %s", g.chain.Topic.Title, stage))
	lines := []string{centerBlock(styleAccent.Render(label), cw), "", centerBlock(status, cw)}
	if g.ev.Defect != "" {
		lines = append(lines, "", centerBlock(styleDim.Render("GOSCII rejected its own draft: "+g.ev.Defect), cw))
	}
	lines = append(lines, "", keyHints(cw, "[esc] hub   [ctrl+c] abort"))
	return screenFrame(g.width, strings.Join(lines, "\n"))
}

func (g GeneratorModel) viewFailed() string {
	cw := contentWidth(g.width)
	lines := []string{centerBlock(styleWarn.Render("⚠  Transmission Failed"), cw), ""}
	if g.lastErr != nil {
		lines = append(lines, styleMuted.Width(cw).Render(g.lastErr.Error()), "")
	}
	lines = append(lines, keyHints(cw, "[R] retry   [M] new mission   [H] hub"))
	return screenFrame(g.width, strings.Join(lines, "\n"))
}

func (g GeneratorModel) viewComplete() string {
	cw := contentWidth(g.width)
	lines := []string{
		centerBlock(stylePass.Render("CHAIN COMPLETE"), cw),
		centerBlock(styleMuted.Render(fmt.Sprintf("%s · %s · %d missions", g.chain.Topic.Title, g.chain.Difficulty, g.chain.Total())), cw),
		"",
	}
	seen := map[string]bool{}
	for _, slug := range g.repairs {
		if seen[slug] {
			continue
		}
		seen[slug] = true
		if sys, ok := engine.FindSystem(slug); ok {
			lines = append(lines, styleText.Render(fmt.Sprintf("  %-16s %s", sys.Name, integrityBar(g.progress.Station.Level(slug)))))
		}
	}
	lines = append(lines, "")
	switch {
	case g.recovered != nil:
		lines = append(lines,
			styleAccent.Render("RECOVERED LOG · "+g.recovered.ID), "",
			storyStyle.Width(cw).Render(g.recovered.Text), "")
	case len(g.progress.Logs) < len(levels.CrewLogs()):
		lines = append(lines, styleDim.Render("Decrypting a recovered sector..."), "")
	}
	if g.usage.Input+g.usage.Output > 0 {
		lines = append(lines, styleDim.Render(fmt.Sprintf("signal spend: %d in / %d out tokens", g.usage.Input, g.usage.Output)), "")
	}
	lines = append(lines, keyHints(cw, "[any key] hub"))
	return screenFrame(g.width, strings.Join(lines, "\n"))
}

func integrityBar(level int) string {
	const width = 20
	fill := level * width / 100
	return styleAccent.Render(strings.Repeat("█", fill)) + styleDim.Render(strings.Repeat("░", width-fill)) + styleMuted.Render(fmt.Sprintf(" %3d%%", level))
}
