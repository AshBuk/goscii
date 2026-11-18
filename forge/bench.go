// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// BenchCase is one cell of the bench matrix.
type BenchCase struct {
	Topic      string            `json:"topic"`
	Beat       string            `json:"beat"`
	Kind       levels.Kind       `json:"kind"`
	Difficulty levels.Difficulty `json:"difficulty"`
}

// BenchRun is the outcome of forging one case.
type BenchRun struct {
	BenchCase
	OK      bool          `json:"ok"`
	Drafts  int           `json:"drafts"`
	Repairs int           `json:"repairs"`
	Elapsed time.Duration `json:"elapsed_ns"`
	Usage   ai.Usage      `json:"usage"`
	Title   string        `json:"title,omitempty"`
	Err     string        `json:"error,omitempty"`
}

// FirstTry reports whether the first answer of the first draft passed.
func (r BenchRun) FirstTry() bool { return r.OK && r.Drafts == 1 && r.Repairs == 0 }

// BenchSummary aggregates a bench.
type BenchSummary struct {
	Missions      int           `json:"missions"`
	FirstTry      int           `json:"first_try"`
	Rescued       int           `json:"rescued"` // passed after repair or a new draft
	Failed        int           `json:"failed"`
	MedianElapsed time.Duration `json:"median_elapsed_ns"`
	TokensIn      int           `json:"tokens_in"`
	TokensOut     int           `json:"tokens_out"`
	RepeatTitles  int           `json:"repeat_titles"`
}

// BenchReport is the JSON a bench writes, for comparing runs over time.
type BenchReport struct {
	Signal  string       `json:"signal"`
	Model   string       `json:"model"`
	At      time.Time    `json:"at"`
	Summary BenchSummary `json:"summary"`
	Runs    []BenchRun   `json:"runs"`
}

// Matrix expands topics, kinds and difficulties into n cases per cell,
// rotating through each topic's beats.
func Matrix(topics []ai.Topic, kinds []levels.Kind, diffs []levels.Difficulty, n int) []BenchCase {
	var out []BenchCase
	for _, t := range topics {
		i := 0
		for _, k := range kinds {
			for _, d := range diffs {
				for range n {
					beats := t.Beats
					if k == levels.Refit {
						beats = slices.DeleteFunc(slices.Clone(beats), func(b ai.Beat) bool { return len(b.Requires) == 0 })
					}
					b := beats[i%len(beats)]
					i++
					out = append(out, BenchCase{Topic: t.Slug, Beat: b.Slug, Kind: k, Difficulty: d})
				}
			}
		}
	}
	return out
}

// Bench forges every case, parallel at a time, and reports each run as it ends.
func Bench(ctx context.Context, sig ai.Provider, cases []BenchCase, parallel int, report func(BenchRun)) BenchReport {
	rep := BenchReport{Signal: sig.Name(), Model: sig.Model(), At: time.Now()}
	runs := make([]BenchRun, len(cases))
	sem := make(chan struct{}, max(1, parallel))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, c := range cases {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			r := benchOne(ctx, sig, c)
			runs[i] = r
			if report != nil {
				mu.Lock()
				report(r)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	rep.Runs = runs
	rep.Summary = summarize(runs)
	return rep
}

func benchOne(ctx context.Context, sig ai.Provider, c BenchCase) BenchRun {
	run := BenchRun{BenchCase: c}
	t, ok := ai.FindTopic(c.Topic)
	if !ok {
		run.Err = "unknown topic"
		return run
	}
	var beat ai.Beat
	for _, b := range t.Beats {
		if b.Slug == c.Beat {
			beat = b
		}
	}
	sys := engine.Systems[len(c.Beat)%len(engine.Systems)]
	started := time.Now()
	res, err := Forge(ctx, sig, Single(t, beat, c.Kind, c.Difficulty, sys), nil)
	run.Elapsed = time.Since(started)
	var fail *Failure
	switch {
	case err == nil:
		run.OK, run.Drafts, run.Repairs, run.Usage, run.Title = true, res.Drafts, res.Repairs, res.Usage, res.Mission.Title
	case errors.As(err, &fail):
		run.Drafts, run.Usage, run.Err = maxDrafts, fail.Usage, err.Error()
	default:
		run.Err = err.Error()
	}
	return run
}

func summarize(runs []BenchRun) BenchSummary {
	s := BenchSummary{Missions: len(runs)}
	var elapsed []time.Duration
	titles := map[string]int{}
	for _, r := range runs {
		s.TokensIn += r.Usage.Input
		s.TokensOut += r.Usage.Output
		elapsed = append(elapsed, r.Elapsed)
		switch {
		case r.FirstTry():
			s.FirstTry++
		case r.OK:
			s.Rescued++
		default:
			s.Failed++
		}
		if r.Title != "" {
			titles[r.Title]++
		}
	}
	for _, n := range titles {
		if n > 1 {
			s.RepeatTitles += n - 1
		}
	}
	if len(elapsed) > 0 {
		slices.Sort(elapsed)
		s.MedianElapsed = elapsed[len(elapsed)/2]
	}
	return s
}
