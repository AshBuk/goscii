// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Generator requests missions from the AI signal, retries until the output validates, then launches the cockpit.
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	width    int
	height   int
}

func NewGeneratorModel(signal ai.Provider, topic ai.Topic, diff levels.Difficulty, p *engine.Progress) GeneratorModel {
	ch := ai.NewMissionChain(topic, diff)
	return GeneratorModel{
		signal:   signal,
		topic:    topic,
		diff:     diff,
		chain:    &ch,
		progress: p,
	}
}

func (g GeneratorModel) Init() tea.Cmd {
	return genAttemptCmd(g.signal, g.buildReq(), 0)
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
		return g, tea.Batch(saveProgressCmd(g.progress), genAttemptCmd(g.signal, g.buildReq(), 0))

	case genAttemptMsg:
		if msg.err != nil {
			g.lastErr = msg.err
			g.state = genStateFailed
			return g, nil
		}
		if msg.valid {
			c := New(msg.mission, msg.tmpl, g.signal, g.chain.Step(), g.chain.Total())
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

	case tea.KeyMsg:
		switch g.state {
		case genStateFailed:
			switch msg.String() {
			case "r", "R":
				g.state = genStateWiring
				g.attempt = 0
				g.lastErr = nil
				return g, genAttemptCmd(g.signal, g.buildReq(), 0)
			case "m", "M":
				return g, func() tea.Msg { return NewMissionMsg{} }
			case "h", "H", "ctrl+c":
				return g, func() tea.Msg { return BackMsg{} }
			}
			return g, nil
		case genStateWiring:
			if msg.String() == "ctrl+c" {
				return g, func() tea.Msg { return BackMsg{} }
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

func (g GeneratorModel) View() string {
	if g.state == genStateCockpit && g.cockpit != nil {
		return g.cockpit.View()
	}
	if g.state == genStateFailed {
		return g.viewFailed()
	}
	return g.viewWiring()
}

func (g GeneratorModel) viewWiring() string {
	var lines []string
	label := fmt.Sprintf("Wiring %s signal for %q", g.diff, g.topic.Title)
	if total := g.chain.Total(); total > 1 {
		label += fmt.Sprintf("  [%d/%d]", g.chain.Step(), total)
	}
	lines = append(lines, selectorAccent.Render(label), "")

	bar := renderGenBar(g.attempt, maxGenAttempts)
	counter := selectorDim.Render(fmt.Sprintf("%d / %d", g.attempt, maxGenAttempts))
	lines = append(lines, selectorMuted.Render("Validating")+"  "+bar+"  "+counter)
	lines = append(lines, "", selectorKeys.Render("[ctrl+c] abort"))
	return selectorFrame(g.width, strings.Join(lines, "\n"))
}

func (g GeneratorModel) viewFailed() string {
	var lines []string
	if g.lastErr != nil {
		lines = append(lines, selectorAccent.Render("Transmission failed: signal error."))
		lines = append(lines, selectorMuted.Render(g.lastErr.Error()), "")
	} else {
		lines = append(lines, selectorAccent.Render("Transmission failed: 8 attempts, no valid mission."))
		lines = append(lines, selectorMuted.Render("The AI could not produce a verifiable response."), "")
	}
	lines = append(lines, selectorKeys.Render("[R] retry   [M] new mission   [H] hub"))
	return selectorFrame(g.width, strings.Join(lines, "\n"))
}

func renderGenBar(filled, total int) string {
	const barWidth = 16
	n := 0
	if total > 0 {
		n = filled * barWidth / total
	}
	bar := strings.Repeat("█", n) + strings.Repeat("░", barWidth-n)
	return genBarStyle.Render(bar)
}

func genAttemptCmd(signal ai.Provider, req ai.Request, attempt int) tea.Cmd {
	return func() tea.Msg {
		m, tmpl, err := signal.Generate(context.Background(), req)
		if err != nil {
			return genAttemptMsg{attempt: attempt, err: err}
		}
		valid := m.Check.Verify(engine.RunCode(tmpl, m.Answer))
		return genAttemptMsg{mission: m, tmpl: tmpl, valid: valid, attempt: attempt}
	}
}

var genBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
