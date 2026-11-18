// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/ai/aitest"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

func order(t *testing.T, topic, beat string, k levels.Kind) Order {
	t.Helper()
	tp, ok := ai.FindTopic(topic)
	if !ok {
		t.Fatalf("topic %q missing", topic)
	}
	for _, b := range tp.Beats {
		if b.Slug == beat {
			return Single(tp, b, k, levels.Medium, engine.Systems[0])
		}
	}
	t.Fatalf("beat %q missing", beat)
	return Order{}
}

func reply(t *testing.T, d draft) string {
	t.Helper()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const sumSolution = `package main

import "fmt"

func main() {
	// === INPUT ===
	readings := []int{3, 4, 5}
	// === END INPUT ===
	// === YOUR CODE HERE ===
	total := 0
	for _, r := range readings {
		total += r
	}
	fmt.Println(total)
	// === END ===
}
`

func sumDraft() draft {
	return draft{
		Title:     "Sum the scrubbers",
		Story:     "Add the scrubber readings and print the total.\nOutput: 12",
		Hints:     []string{"Loop over the readings.", "for _, r := range readings"},
		Solution:  sumSolution,
		AltInputs: []string{"readings := []int{1, 1}", "readings := []int{10, 20, 30, 40}", "readings := []int{4, 4, 4}"},
	}
}

func TestForgeRepairComputesTruth(t *testing.T) {
	sig := aitest.New(reply(t, sumDraft()))
	res, err := Forge(context.Background(), sig, order(t, "control-flow", "range-string", levels.Repair), nil)
	if err != nil {
		t.Fatal(err)
	}
	m := res.Mission
	if m.Check.StdoutEquals != "12" {
		t.Fatalf("expected computed output 12, got %q", m.Check.StdoutEquals)
	}
	if strings.Contains(m.Story, "Output:") {
		t.Fatalf("Output line should be stripped: %q", m.Story)
	}
	if len(m.Check.Cases) != 2 {
		t.Fatalf("want 2 cases (third repeats the output), got %+v", m.Check.Cases)
	}
	if strings.Contains(res.Template, "total := 0") {
		t.Fatalf("template leaks the answer:\n%s", res.Template)
	}
	if v := engine.Judge(res.Template, m.Answer, m.Check); !v.Passed {
		t.Fatalf("reference must pass its own check: %+v", v)
	}
	if v := engine.Judge(res.Template, `fmt.Println(12)`, m.Check); v.Passed {
		t.Fatal("hardcoded answer must fail the cases")
	}
}

func TestForgeRepairsCompileError(t *testing.T) {
	broken := sumDraft()
	broken.Solution = strings.Replace(sumSolution, "total += r", "total += missing", 1)
	sig := aitest.New(reply(t, broken), reply(t, sumDraft()))
	var events []Event
	res, err := Forge(context.Background(), sig, order(t, "control-flow", "range-string", levels.Repair), func(e Event) { events = append(events, e) })
	if err != nil {
		t.Fatal(err)
	}
	if res.Repairs != 1 || res.Drafts != 1 {
		t.Fatalf("want 1 repair in 1 draft, got %d repairs %d drafts", res.Repairs, res.Drafts)
	}
	second := sig.Prompts[1]
	if len(second.Turns) != 3 {
		t.Fatalf("repair must continue the conversation, got %d turns", len(second.Turns))
	}
	feedback := second.Turns[2].Text
	if !strings.Contains(feedback, "does not compile") || !strings.Contains(feedback, "missing") {
		t.Fatalf("repair message must carry the compiler error, got:\n%s", feedback)
	}
	if !hasStage(events, Recalibrating) {
		t.Fatal("expected a recalibrating event")
	}
}

func hasStage(events []Event, s Stage) bool {
	for _, e := range events {
		if e.Stage == s {
			return true
		}
	}
	return false
}

func TestForgeRejectsHollowScaffold(t *testing.T) {
	d := sumDraft()
	d.Solution = `package main

import "fmt"

func main() {
	// === YOUR CODE HERE ===
	for range 1 {
	}
	// === END ===
	fmt.Println(12)
}
`
	_, def := assay(reply(t, d), order(t, "control-flow", "range-string", levels.Repair))
	if def == nil || !strings.Contains(def.Rule, "by itself") {
		t.Fatalf("want hollow-scaffold defect, got %v", def)
	}
}

func TestForgeEnforcesBeatRule(t *testing.T) {
	d := sumDraft()
	o := order(t, "select", "two-chans", levels.Repair)
	_, def := assay(reply(t, d), o)
	if def == nil || !strings.Contains(def.Rule, "idiom") {
		t.Fatalf("a select beat solved without select must be rejected, got %v", def)
	}
}

func TestForgeFault(t *testing.T) {
	d := draft{
		Title:    "Manifest crash",
		Story:    "The manifest counter panics on boot.",
		Hints:    []string{"A map needs to exist before you write to it.", "make(map[string]int)"},
		Solution: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\tcounts := make(map[string]int)\n\tcounts[\"crew\"]++\n\tfmt.Println(counts[\"crew\"])\n\t// === END ===\n}\n",
		Faulty:   "var counts map[string]int\ncounts[\"crew\"]++\nfmt.Println(counts[\"crew\"])",
		Fault:    "writes to a nil map",
	}
	got, def := assay(reply(t, d), order(t, "maps", "frequency", levels.Fault))
	if def != nil {
		t.Fatal(def)
	}
	if got.mission.Seed == "" || got.mission.Fault == "" {
		t.Fatalf("fault mission needs seed and fault: %+v", got.mission)
	}
	if v := engine.Judge(got.template, got.mission.Seed, got.mission.Check); v.Passed {
		t.Fatal("the seed must fail")
	}

	harmless := d
	harmless.Faulty = "counts := map[string]int{}\ncounts[\"crew\"]++\nfmt.Println(counts[\"crew\"])"
	if _, def := assay(reply(t, harmless), order(t, "maps", "frequency", levels.Fault)); def == nil || !strings.Contains(def.Rule, "harmless") {
		t.Fatalf("want harmless-fault defect, got %v", def)
	}
}

func TestForgeDecode(t *testing.T) {
	d := draft{
		Title:    "Unwind log",
		Story:    "Predict the shutdown order.",
		Hints:    []string{"Deferred calls run last-in, first-out."},
		Solution: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfor i := range 3 {\n\t\tdefer fmt.Println(i)\n\t}\n}\n",
	}
	got, def := assay(reply(t, d), order(t, "functions", "defer-order", levels.Decode))
	if def != nil {
		t.Fatal(def)
	}
	if got.mission.Check.StdoutEquals != "2\n1\n0" || !got.mission.Check.Prediction {
		t.Fatalf("decode truth must come from a run: %+v", got.mission.Check)
	}
	if v := engine.Judge(got.template, "2\n1\n0\n", got.mission.Check); !v.Passed {
		t.Fatalf("correct prediction must pass: %+v", v)
	}
	if v := engine.Judge(got.template, "0\n1\n2", got.mission.Check); v.Passed {
		t.Fatal("wrong prediction must fail")
	}
}

func TestForgeRefit(t *testing.T) {
	d := draft{
		Title:    "Beacon call sign",
		Story:    "Rebuild the call sign with a builder.",
		Hints:    []string{"Concatenation allocates every time."},
		Solution: "package main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\nfunc main() {\n\tparts := []string{\"GO\", \"SC\", \"II\"}\n\t// === YOUR CODE HERE ===\n\tvar b strings.Builder\n\tfor _, p := range parts {\n\t\tb.WriteString(p)\n\t}\n\tfmt.Println(b.String())\n\t// === END ===\n}\n",
		Seed:     "s := \"\"\nfor _, p := range parts {\n\ts += p\n}\nfmt.Println(s)\n_ = strings.ToUpper",
	}
	got, def := assay(reply(t, d), order(t, "strings", "builder", levels.Refit))
	if def != nil {
		t.Fatal(def)
	}
	v := engine.Judge(got.template, got.mission.Seed, got.mission.Check)
	if v.Passed || !strings.Contains(v.Failure, "off-protocol") {
		t.Fatalf("seed prints the right output but must fail the protocol rule: %+v", v)
	}
}

func TestForgeGivesUpAfterDrafts(t *testing.T) {
	bad := reply(t, draft{Title: "x"})
	var texts []string
	for range maxDrafts * (maxRepairs + 1) {
		texts = append(texts, bad)
	}
	sig := aitest.New(texts...)
	_, err := Forge(context.Background(), sig, order(t, "control-flow", "range-string", levels.Repair), nil)
	var f *Failure
	if !errors.As(err, &f) || len(f.Defects) != maxDrafts {
		t.Fatalf("want failure with %d drafts, got %v", maxDrafts, err)
	}
}

func TestForgeStopsOnTransportError(t *testing.T) {
	sig := aitest.New()
	sig.Then(aitest.Step{Err: ai.ErrKeyRejected})
	_, err := Forge(context.Background(), sig, order(t, "control-flow", "range-string", levels.Repair), nil)
	if !errors.Is(err, ai.ErrKeyRejected) || sig.Calls() != 1 {
		t.Fatalf("a rejected key must abort at once, got %v after %d calls", err, sig.Calls())
	}
}

func TestEveryBeatRuleIsKnown(t *testing.T) {
	for _, tp := range ai.StdlibTopics {
		if len(tp.Beats) == 0 {
			t.Errorf("topic %s has no beats", tp.Slug)
		}
		for _, b := range tp.Beats {
			for _, r := range append(append([]string{}, b.Requires...), b.Forbids...) {
				if !engine.ValidRule(r) {
					t.Errorf("%s/%s: unknown rule %q", tp.Slug, b.Slug, r)
				}
			}
		}
	}
}

func TestChainRotatesBeatsAndSystems(t *testing.T) {
	tp, _ := ai.FindTopic("maps")
	c := NewChain(tp, levels.Medium, "")
	p := &engine.Progress{}
	beats, systems := map[string]bool{}, map[string]bool{}
	for !c.Done() {
		o := c.Next(p)
		if again := c.Next(p); again.Beat.Slug != o.Beat.Slug {
			t.Fatal("Next must be stable for the same step")
		}
		beats[o.Beat.Slug] = true
		systems[o.System.Slug] = true
		c.Record(Entry{Title: "t", Beat: o.Beat.Slug, System: o.System.Slug})
	}
	if len(beats) != c.Total() || len(systems) != c.Total() {
		t.Fatalf("want %d distinct beats and systems, got %d and %d", c.Total(), len(beats), len(systems))
	}
}

func TestSchemaIsStrict(t *testing.T) {
	for _, k := range levels.Kinds {
		s := schema(k)
		props := s["properties"].(map[string]any)
		if len(s["required"].([]string)) != len(props) {
			t.Errorf("%s: every property must be required", k)
		}
	}
}

func TestRefitWithoutRulesPlansRepair(t *testing.T) {
	c := NewChain(ai.OpenTopic("worker pools"), levels.Easy, levels.Refit)
	if k := c.Next(&engine.Progress{}).Kind; k != levels.Repair {
		t.Fatalf("an open channel has no rule to refit toward; want repair, got %s", k)
	}
	mixed := NewChain(ai.OpenTopic("worker pools"), levels.Hard, "")
	for range mixed.Total() {
		if o := mixed.Next(&engine.Progress{}); o.Kind == levels.Refit {
			t.Fatal("mixed rotation must skip refit on an open channel")
		}
		mixed.Record(Entry{Title: "x"})
	}
}
