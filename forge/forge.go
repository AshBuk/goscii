// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/levels"
)

const (
	maxDrafts  = 3
	maxRepairs = 2 // repair rounds per draft
)

// Stage is a step of the forging pipeline, reported as it starts.
type Stage int

const (
	Transmitting  Stage = iota // waiting for the signal
	TestFiring                 // building and running the draft
	Recalibrating              // sending diagnostics back for a repair
)

func (s Stage) String() string {
	switch s {
	case TestFiring:
		return "test-firing repair"
	case Recalibrating:
		return "recalibrating"
	default:
		return "decoding transmission"
	}
}

// Event reports progress to whoever is watching the forge.
type Event struct {
	Stage  Stage
	Draft  int    // 1-based
	Repair int    // repair round within the draft, 0 for the first answer
	Defect string // the rule that sent the draft back, on Recalibrating
}

// Result is a verified mission and what it cost to forge.
type Result struct {
	Mission  *levels.Mission
	Template string
	Usage    ai.Usage
	Drafts   int
	Repairs  int // repair rounds across all drafts
	Elapsed  time.Duration
}

// Failure reports a forge that ran out of drafts.
type Failure struct {
	Defects []string // last defect of each draft
	Usage   ai.Usage
}

func (f *Failure) Error() string {
	if len(f.Defects) == 0 {
		return "GOSCII could not forge a valid mission"
	}
	return fmt.Sprintf("GOSCII could not forge a valid mission after %d drafts; last fault: %s", len(f.Defects), f.Defects[len(f.Defects)-1])
}

// Forge drafts a mission, verifies it, and sends diagnostics back until it
// passes: up to maxRepairs repair rounds per draft and maxDrafts drafts.
// Transport errors abort at once; the transport already retried what it could.
func Forge(ctx context.Context, sig ai.Provider, o Order, report func(Event)) (*Result, error) {
	if report == nil {
		report = func(Event) {}
	}
	started := time.Now()
	res := &Result{}
	fail := &Failure{}
	prompt := ai.Prompt{
		System: systemPrompt(o.Kind, o.Difficulty),
		Schema: schema(o.Kind),
		Effort: "medium",
	}
	for draftNo := 1; draftNo <= maxDrafts; draftNo++ {
		res.Drafts = draftNo
		prompt.Turns = []ai.Turn{ai.User(orderMessage(o))}
		var last string
		for round := 0; round <= maxRepairs; round++ {
			report(Event{Stage: Transmitting, Draft: draftNo, Repair: round})
			reply, err := sig.Complete(ctx, prompt)
			res.Usage.Add(reply.Usage)
			fail.Usage.Add(reply.Usage)
			if errors.Is(err, ai.ErrTruncated) || errors.Is(err, ai.ErrRefused) {
				last = err.Error()
				break // start a fresh draft
			}
			if err != nil {
				return nil, err
			}
			report(Event{Stage: TestFiring, Draft: draftNo, Repair: round})
			got, def := assay(reply.Text(), o)
			if def == nil {
				res.Mission = got.mission
				res.Template = got.template
				res.Elapsed = time.Since(started)
				return res, nil
			}
			last = def.Error()
			if round == maxRepairs {
				break
			}
			res.Repairs++
			report(Event{Stage: Recalibrating, Draft: draftNo, Repair: round + 1, Defect: def.Rule})
			prompt.Turns = append(prompt.Turns, reply.Turn, ai.User(repairMessage(def)))
		}
		fail.Defects = append(fail.Defects, last)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return nil, fail
}
