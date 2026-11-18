// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package forge turns a mission order into a verified mission. The harness owns
// state, randomness and truth: it picks the idiom and the setting, computes the
// expected output by running the reference, and sends the signal precise
// diagnostics when a draft fails. The signal owns only text and code.
package forge

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// Order is everything the signal needs to write one mission.
type Order struct {
	Topic      ai.Topic
	Beat       ai.Beat
	Kind       levels.Kind
	Difficulty levels.Difficulty
	System     engine.System
	Dossier    string // pilot record from progress
	Brief      string // chain history
	Wish       string // the player's own words, from the open channel
	Carry      string // incident mode: the player's previous program, reused as scaffold
	Repaired   int    // missions completed overall; sets GOSCII's voice
}

// Entry records one completed mission in a chain.
type Entry struct {
	Title      string
	Story      string
	Beat       string
	System     string
	PlayerCode string
	Program    string // the full program the player's code produced
}

// Chain plans a sequence of related missions for one topic and difficulty.
type Chain struct {
	Topic      ai.Topic
	Difficulty levels.Difficulty
	Kind       levels.Kind // empty means mixed
	Wish       string
	Incident   bool // each step builds on the previous program
	entries    []Entry
	planned    []Order
	maxLen     int
	rng        *rand.Rand
}

// NewChain starts a chain. kind may be empty for a mixed rotation.
func NewChain(t ai.Topic, d levels.Difficulty, kind levels.Kind) *Chain {
	return &Chain{
		Topic:      t,
		Difficulty: d,
		Kind:       kind,
		maxLen:     chainLength(d),
		rng:        rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

func chainLength(d levels.Difficulty) int {
	switch d {
	case levels.Medium:
		return 6
	case levels.Hard, levels.Survival:
		return 12
	default:
		return 3
	}
}

// Record adds a completed mission to the chain history.
func (c *Chain) Record(e Entry) { c.entries = append(c.entries, e) }

// Done reports whether the chain has reached its mission limit.
func (c *Chain) Done() bool { return len(c.entries) >= c.maxLen }

// Step returns the current mission number (1-based).
func (c *Chain) Step() int { return len(c.entries) + 1 }

// Total returns the total number of missions in this chain.
func (c *Chain) Total() int { return c.maxLen }

// Final reports whether the current step is the last: a capstone without hints
// that combines the idioms of the earlier steps.
func (c *Chain) Final() bool { return c.Step() == c.maxLen && c.maxLen > 1 }

// mixed is the kind rotation for a chain without a fixed kind.
var mixed = []levels.Kind{levels.Repair, levels.Decode, levels.Fault, levels.Repair, levels.Refit, levels.Fault}

// Next plans the order for the current step. Calling it again for the same
// step returns the same plan, so a retry rewrites the same mission.
func (c *Chain) Next(p *engine.Progress) Order {
	step := c.Step()
	if len(c.planned) >= step {
		return c.planned[step-1]
	}
	kind := c.Kind
	if kind == "" {
		kind = mixed[(step-1)%len(mixed)]
	}
	if c.Incident {
		kind = levels.Repair
	}
	beat := c.pickBeat(p, kind)
	o := Order{
		Topic:      c.Topic,
		Beat:       beat,
		Kind:       kind,
		Difficulty: c.Difficulty,
		System:     c.pickSystem(p),
		Dossier:    dossier(p, c.Topic),
		Brief:      c.brief(),
		Wish:       c.Wish,
		Repaired:   p.Repaired(),
	}
	if c.Incident && len(c.entries) > 0 {
		last := c.entries[len(c.entries)-1]
		o.Carry = last.Program
		o.System, _ = engine.FindSystem(last.System)
	}
	c.planned = append(c.planned, o)
	return o
}

// Peek plans the step after the current one as if e had been recorded, so the
// next mission can be forged while the current one is still in play. The plan
// sticks: once e's real entry is recorded, Next returns the peeked order.
func (c *Chain) Peek(p *engine.Progress, e Entry) (Order, bool) {
	if c.Incident || c.Step() >= c.maxLen {
		return Order{}, false
	}
	c.entries = append(c.entries, e)
	o := c.Next(p)
	c.entries = c.entries[:len(c.entries)-1]
	return o, true
}

// pickBeat prefers beats the pilot has practised least, skips beats already in
// this chain, and leans toward later beats on hard difficulties. Refit needs a
// beat with a construct rule, or there is nothing to refit toward.
func (c *Chain) pickBeat(p *engine.Progress, kind levels.Kind) ai.Beat {
	used := map[string]bool{}
	for _, o := range c.planned {
		used[o.Beat.Slug] = true
	}
	beats := c.Topic.Beats
	best, bestScore := -1, 0
	for i, b := range beats {
		if used[b.Slug] && len(used) < len(beats) {
			continue
		}
		if kind == levels.Refit && len(b.Requires) == 0 {
			continue
		}
		done, _ := p.BeatRuns(c.Topic.Slug, b.Slug)
		order := i
		if c.Difficulty == levels.Hard || c.Difficulty == levels.Survival {
			order = len(beats) - i
		}
		score := done*100 + order
		if best == -1 || score < bestScore {
			best, bestScore = i, score
		}
	}
	if best == -1 {
		return beats[c.rng.IntN(len(beats))]
	}
	return beats[best]
}

// pickSystem chooses a station system not used earlier in the chain, preferring
// damaged ones so repairs land where the station needs them.
func (c *Chain) pickSystem(p *engine.Progress) engine.System {
	used := map[string]bool{}
	for _, o := range c.planned {
		used[o.System.Slug] = true
	}
	var pool []engine.System
	for _, s := range engine.Systems {
		if !used[s.Slug] {
			pool = append(pool, s)
		}
	}
	if len(pool) == 0 {
		pool = engine.Systems
	}
	var station engine.Station
	if p != nil {
		station = p.Station
	}
	slices.SortStableFunc(pool, func(a, b engine.System) int { return station.Level(a.Slug) - station.Level(b.Slug) })
	// pick among the three most damaged so chains do not always open on the same system
	return pool[c.rng.IntN(min(3, len(pool)))]
}

func (c *Chain) brief() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Mission %d of %d in this chain.", c.Step(), c.maxLen)
	if c.Final() {
		sb.WriteString(" This is the final mission: combine the idioms of the earlier missions in one repair.")
	}
	if len(c.entries) == 0 {
		return sb.String()
	}
	sb.WriteString("\nRecent missions the astronaut solved:\n")
	start := max(0, len(c.entries)-3)
	for i, e := range c.entries[start:] {
		fmt.Fprintf(&sb, "%d. %q (%s, idiom %s). Astronaut wrote:\n%s\n", start+i+1, e.Title, e.System, e.Beat, e.PlayerCode)
	}
	sb.WriteString("Continue the same narrative thread; reference what the astronaut already fixed.")
	return sb.String()
}

func dossier(p *engine.Progress, t ai.Topic) string {
	if p == nil {
		return ""
	}
	var done, struggled []string
	for _, b := range t.Beats {
		n, worst := p.BeatRuns(t.Slug, b.Slug)
		if n > 0 {
			done = append(done, b.Slug)
		}
		if worst >= 4 {
			struggled = append(struggled, b.Slug)
		}
	}
	total := len(p.Topics[t.Slug].Completed)
	if total == 0 {
		return "Pilot record: first mission on this topic."
	}
	s := fmt.Sprintf("Pilot record: %d missions completed on this topic.", total)
	if len(done) > 0 {
		s += " Practised idioms: " + strings.Join(done, ", ") + "."
	}
	if len(struggled) > 0 {
		s += " Needed many runs on: " + strings.Join(struggled, ", ") + "; keep those parts gentle."
	}
	return s
}

// Single plans one standalone order, used by the bench and forge commands.
func Single(t ai.Topic, b ai.Beat, k levels.Kind, d levels.Difficulty, sys engine.System) Order {
	return Order{Topic: t, Beat: b, Kind: k.Or(), Difficulty: d, System: sys}
}
