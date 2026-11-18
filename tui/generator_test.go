// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/ai/aitest"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

var errTest = errors.New("signal lost")

func newTestGenerator(t *testing.T) GeneratorModel {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	topic, _ := ai.FindTopic("maps")
	chain := forge.NewChain(topic, levels.Easy, levels.Repair)
	return NewGeneratorModel(aitest.New(), chain, engine.Config{}, &engine.Progress{})
}

// firstMsg runs the first command of a batch and returns its message.
func firstMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return batch[0]()
	}
	return msg
}

func TestFirstForgeReplyReachesTheKeptModel(t *testing.T) {
	g := newTestGenerator(t)
	msg := firstMsg(t, g.Init()) // the exhausted script fails the forge at once
	for {
		if _, ok := msg.(forgeDoneMsg); ok {
			break
		}
		ev := msg.(forgeEventMsg)
		msg = waitForge(ev.ch, ev.gen)()
	}
	m, _ := g.Update(msg)
	if m.(GeneratorModel).state != genFailed {
		t.Fatal("the reply to Init's forge must reach the model the hub keeps")
	}
}

func TestStaleForgeReplyIsDropped(t *testing.T) {
	g := newTestGenerator(t)
	m, _ := g.Update(forgeDoneMsg{gen: nextID(), err: errTest})
	if m.(GeneratorModel).state != genForging {
		t.Fatal("a reply from another forge must not end this one")
	}
	m, _ = g.Update(forgeDoneMsg{gen: nextID(), err: errTest, prefetch: true})
	if m.(GeneratorModel).state != genForging {
		t.Fatal("a stray prefetch reply must not touch this chain")
	}
}

func TestLateRunWithoutCockpitIsDropped(t *testing.T) {
	g := newTestGenerator(t)
	if _, cmd := g.Update(runDoneMsg{id: nextID(), verdict: engine.Verdict{Passed: true}}); cmd != nil {
		t.Fatal("a run from a closed mission must be ignored")
	}
}

func TestHubBackStopsChildWork(t *testing.T) {
	g := newTestGenerator(t)
	ctx, _ := g.work.context(time.Minute)
	h := HubModel{child: g}
	m, _ := h.Update(BackMsg{})
	if ctx.Err() == nil {
		t.Fatal("leaving the screen must cancel its forges")
	}
	if m.(HubModel).child != nil {
		t.Fatal("hub must drop the child")
	}
}

func launchTestMission(t *testing.T, g GeneratorModel) GeneratorModel {
	t.Helper()
	tmpl := "package main\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\n\t// === END ===\n}\n"
	m := &levels.Mission{ID: "m1", Beat: "comma-ok", Kind: levels.Repair, Difficulty: levels.Easy, System: "dock", Answer: "x", Check: engine.CheckRule{StdoutEquals: "x"}}
	launched, _ := g.launch(&forge.Result{Mission: m, Template: tmpl})
	return launched.(GeneratorModel)
}

func pass(t *testing.T, g GeneratorModel) GeneratorModel {
	t.Helper()
	m, _ := g.Update(runDoneMsg{id: g.cockpit.work.id, verdict: engine.Verdict{Passed: true}})
	return m.(GeneratorModel)
}

func TestPassCreditsBeforeNext(t *testing.T) {
	g := pass(t, launchTestMission(t, newTestGenerator(t)))
	p := g.progress
	if len(p.Topics["maps"].Completed) != 1 || p.Station.Level("dock") <= (engine.Station{}).Level("dock") {
		t.Fatalf("a pass must record completion and repair at once: %+v", p)
	}
	g = pass(t, g)
	if len(p.Topics["maps"].Completed) != 1 {
		t.Fatal("a second run of a passed mission must not credit twice")
	}
	m, _ := g.Update(NextMsg{})
	if len(m.(GeneratorModel).progress.Topics["maps"].Completed) != 1 {
		t.Fatal("ctrl+n must not credit again")
	}
}

func TestFinalPassUnlocksCrewLog(t *testing.T) {
	g := newTestGenerator(t)
	for g.chain.Step() < g.chain.Total() {
		g.chain.Next(g.progress)
		g.chain.Record(forge.Entry{Title: "done"})
	}
	g = pass(t, launchTestMission(t, g))
	if len(g.progress.Logs) != 1 || g.recovered == nil || g.progress.Logs[0].Text == "" {
		t.Fatalf("the chain's log must be stored before any key: %+v", g.progress.Logs)
	}
}

func TestAdventurePassMovesCheckpoint(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := &engine.Progress{}
	r := NewAdventureRunner("onboarding", p).(*adventureRunner)
	r.Update(runDoneMsg{id: r.cockpit.work.id, verdict: engine.Verdict{Passed: true}})
	if p.AdventureDone != 1 || p.AdventureCheckpoint != r.paths[1] {
		t.Fatalf("a pass must save the next checkpoint at once: done=%d checkpoint=%q", p.AdventureDone, p.AdventureCheckpoint)
	}
}

func TestAdventurePassRepairsItsSystem(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := &engine.Progress{AdventureCheckpoint: "adventures/onboarding/03_types_decode"}
	r := NewAdventureRunner("onboarding", p).(*adventureRunner)
	before := p.Station.Level("power")
	r.Update(runDoneMsg{id: r.cockpit.work.id, verdict: engine.Verdict{Passed: true}})
	if p.Station.Level("power") <= before {
		t.Fatal("a bundled mission with a system must repair it on pass")
	}
}
