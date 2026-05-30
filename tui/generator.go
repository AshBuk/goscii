// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Generator requests missions from the AI signal, retries until the output validates, then launches the cockpit.
package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

const maxGenAttempts = 8

type genState int

const (
	genStateWiring genState = iota
	genStateCockpit
	genStateFailed
)

type genAttemptMsg struct {
	mission *levels.Mission
	tmpl    string
	valid   bool
	attempt int
	err     error
}

// GeneratorModel shows progress during AI signal generation and owns the Cockpit once ready.
type GeneratorModel struct {
	state    genState
	signal   ai.Provider
	topic    ai.Topic
	diff     levels.Difficulty
	chain    *ai.MissionChain
	progress *engine.Progress
	attempt  int
	lastErr  error
	mission  *levels.Mission
	cockpit  *Cockpit
	spin     spinner.Model
	width    int
	height   int
}

func NewGeneratorModel(signal ai.Provider, topic ai.Topic, diff levels.Difficulty, p *engine.Progress) GeneratorModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)
	ch := ai.NewMissionChain(topic, diff)
	return GeneratorModel{
		signal:   signal,
		topic:    topic,
		diff:     diff,
		chain:    &ch,
		progress: p,
		spin:     sp,
	}
}

func (g GeneratorModel) Init() tea.Cmd {
	return tea.Batch(genAttemptCmd(g.signal, g.buildReq(), 0), g.spin.Tick)
}

func (g GeneratorModel) buildReq() ai.Request {
	return ai.Request{
		Topic:      g.topic,
		Difficulty: g.diff,
		Extra:      g.chain.Brief(),
	}
}

func (g GeneratorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocyclo // bubbletea "one switch" updates
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height
		if g.cockpit != nil {
			newC, cmd := g.cockpit.Update(msg)
			c := newC.(Cockpit)
			g.cockpit = &c
			return g, cmd
		}
		return g, nil

	case BackMsg:
		return g, func() tea.Msg { return BackMsg{} }

	case NextMsg:
		if g.cockpit == nil {
			return g, nil
		}
		g.chain.Record(g.mission.Story, g.cockpit.PlayerCode())
		g.progress.RecordCompletion(g.topic.Slug, g.mission.ID, string(g.diff))
		if g.chain.Done() {
			return g, tea.Batch(saveProgressCmd(g.progress), func() tea.Msg { return BackMsg{} })
		}
		g.state = genStateWiring
		g.attempt = 0
		g.lastErr = nil
		g.cockpit = nil
		return g, tea.Batch(saveProgressCmd(g.progress), genAttemptCmd(g.signal, g.buildReq(), 0), g.spin.Tick)

	case spinner.TickMsg:
		if g.state == genStateWiring {
			var cmd tea.Cmd
			g.spin, cmd = g.spin.Update(msg)
			return g, cmd
		}
		return g, nil

	case genAttemptMsg:
		if msg.err != nil {
			g.lastErr = msg.err
			g.state = genStateFailed
			return g, nil
		}
		if msg.valid {
			c := New(msg.mission, msg.tmpl, g.signal, g.chain.Step(), g.chain.Total(), g.topic.Slug)
			sized, _ := c.Update(tea.WindowSizeMsg{Width: g.width, Height: g.height})
			c = sized.(Cockpit)
			g.mission = msg.mission
			g.cockpit = &c
			g.state = genStateCockpit
			return g, c.Init()
		}
		if msg.attempt < maxGenAttempts-1 {
			g.attempt = msg.attempt + 1
			return g, genAttemptCmd(g.signal, g.buildReq(), g.attempt)
		}
		g.state = genStateFailed
		return g, nil

	case tea.KeyPressMsg:
		switch g.state {
		case genStateFailed:
			switch msg.String() {
			case "r", "R":
				g.state = genStateWiring
				g.attempt = 0
				g.lastErr = nil
				return g, tea.Batch(genAttemptCmd(g.signal, g.buildReq(), 0), g.spin.Tick)
			case "m", "M":
				return g, func() tea.Msg { return NewMissionMsg{} }
			case "h", "H":
				return g, func() tea.Msg { return BackMsg{} }
			case "ctrl+c":
				return g, tea.Quit
			}
			return g, nil
		case genStateWiring:
			if msg.String() == "ctrl+c" {
				return g, tea.Quit
			}
			return g, nil
		}
	}

	if g.state == genStateCockpit && g.cockpit != nil {
		newC, cmd := g.cockpit.Update(msg)
		c := newC.(Cockpit)
		g.cockpit = &c
		return g, cmd
	}
	return g, nil
}

func (g GeneratorModel) View() tea.View {
	if g.state == genStateCockpit && g.cockpit != nil {
		return g.cockpit.View()
	}
	if g.state == genStateFailed {
		return tea.NewView(g.viewFailed())
	}
	return tea.NewView(g.viewWiring())
}

func (g GeneratorModel) viewWiring() string {
	cw := contentWidth(g.width)
	label := fmt.Sprintf("Establishing %s signal...", g.diff)
	if total := g.chain.Total(); total > 1 {
		label += fmt.Sprintf("  [%d/%d]", g.chain.Step(), total)
	}
	status := g.spin.View() + " " + styleMuted.Render(
		fmt.Sprintf("%q · attempt %d / %d", g.topic.Title, g.attempt, maxGenAttempts),
	)
	var lines []string
	lines = append(lines, centerBlock(styleAccent.Render(label), cw), "")
	lines = append(lines, centerBlock(status, cw))
	lines = append(lines, "", keyHints(cw, "[ctrl+c] abort"))
	return screenFrame(g.width, strings.Join(lines, "\n"))
}

func (g GeneratorModel) viewFailed() string {
	cw := contentWidth(g.width)
	var lines []string
	lines = append(lines, centerBlock(styleWarn.Render("⚠  Transmission Failed"), cw), "")
	if g.lastErr != nil {
		lines = append(lines, styleMuted.Render(g.lastErr.Error()), "")
	} else {
		lines = append(lines, styleMuted.Render("8 attempts exhausted — AI could not produce a valid mission."), "")
	}
	lines = append(lines, keyHints(cw, "[R] retry   [M] new mission   [H] hub"))
	return screenFrame(g.width, strings.Join(lines, "\n"))
}

func genAttemptCmd(signal ai.Provider, req ai.Request, attempt int) tea.Cmd {
	return func() tea.Msg {
		m, tmpl, err := signal.Generate(context.Background(), req)
		if err != nil {
			return genAttemptMsg{attempt: attempt, err: err}
		}
		valid := engine.TemplateWellFormed(tmpl) && m.Check.Verify(engine.RunCode(tmpl, m.Answer))
		return genAttemptMsg{mission: m, tmpl: tmpl, valid: valid, attempt: attempt}
	}
}
