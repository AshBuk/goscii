// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"fmt"
	"strings"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

const programRules = `Program rules for "solution":
- One file, package main, standard library only. It must build with go build and exit with status 0.
- Output is deterministic: no time.Now, no randomness, no map iteration order and no goroutine scheduling order may reach the output.
- Never reach the network. Touch files only under os.TempDir().
- Keep it small and readable: a mission is a few minutes of work.`

const regionRules = `- Mark the astronaut's region with these two lines, exactly once each:
    // === YOUR CODE HERE ===
    // === END ===
  The code between them is the reference answer. Everything outside is the read-only scaffold. The region is 3 to 25 lines.
- Marker placement: for tasks that compute or print, put the markers inside func main(). For tasks that declare functions, types or methods, put the markers at package level; then either the region also contains func main, or the scaffold after // === END === has a func main that exercises the astronaut's declarations.
- The scaffold must parse on its own: never split a block, an import group or a call across a marker.
- With the region emptied, the program must fail to build or print something else. The scaffold never prints the answer by itself.
- Input ownership: when the task processes data, declare the concrete input values in the scaffold above the region, between these two lines:
    // === INPUT ===
    // === END INPUT ===
  Only plain variable declarations with literal values go there. In "alt_inputs" give two alternative bodies for that block: same variable names and types, different values, chosen so that a hardcoded answer fails. If the task has no input data, omit the input block and send an empty alt_inputs list.`

const textRules = `Text rules:
- "title": 2 to 5 words.
- "hints": 2 or 3 strings. The first is conceptual, the last is a concrete syntax nudge. Never the full answer.
- "story" never contains the expected output or the solution code, and never ends with an "Output:" line: the cockpit shows the target output itself.`

var kindRules = map[levels.Kind]string{
	levels.Repair: `Mission kind: repair. The astronaut writes the missing code in the region.`,
	levels.Fault: `Mission kind: fault. The station's code looks right but a fault is planted in it.
- "solution" holds the correct program with the region markers around the part that contains the fault.
- "faulty" holds the same region with exactly one planted fault. It must compile. When run it must panic, deadlock, or print the wrong output.
- Plant a real Go pitfall that fits the idiom: writing to a nil map, a shadowed err, an off-by-one in a slice bound, a value receiver that loses a mutation, a channel never closed so range deadlocks, a deferred call whose argument is evaluated too early, integer division truncating, append aliasing a shared backing array, a missing break out of a loop.
- "faulty" and the reference region differ in one to three lines. Do not mark the fault with a comment.
- "fault": one sentence for GOSCII's internal log naming the fault. The astronaut does not see it.
- The story describes the symptom, not the cause.`,
	levels.Decode: `Mission kind: decode. The astronaut reads a recovered program and predicts its exact output.
- "solution" is the complete program with NO region markers and no input block.
- The output is 1 to 8 short lines, and predicting it requires understanding the idiom: defer order, slice aliasing, closure capture, map zero values, rune versus byte counts, method sets, shadowing.
- No randomness and nothing that looks like a trick of formatting; the difficulty is in the Go semantics.
- The story asks the astronaut to work out what the program prints before GOSCII runs it.`,
	levels.Refit: `Mission kind: refit. Working code must be rewritten to station protocol.
- "solution" holds the target version: the region uses the idiom.
- "seed" holds the same region written the clumsy way, without the idiom, producing exactly the same output. The astronaut starts from the seed.
- The story says what protocol the code must follow and why it matters aboard.`,
}

var storyRules = map[levels.Difficulty]string{
	levels.Easy:     "Story for easy: plain instruction. State exactly what to declare and what to print. A light sci-fi frame is fine but the task must be unambiguous.",
	levels.Medium:   "Story for medium: 2 to 4 sentences of scenario, with the goal still clear.",
	levels.Hard:     "Story for hard: terse and in GOSCII's voice. The difficulty comes from the Go concept (edge cases, composition, idiomatic patterns), never from an unclear task.",
	levels.Survival: "Story for survival: corrupted and fragmentary. The astronaut sees no target output and no hints, so the story must still contain every fact needed to solve it, scattered through the noise.",
}

// systemPrompt is stable per kind and difficulty so providers can cache it.
func systemPrompt(k levels.Kind, d levels.Difficulty) string {
	parts := []string{
		ai.Persona,
		"You write the station's repair jobs as Go exercises. The harness that runs you has already chosen the Go idiom, the station system and the mission kind. You write the mission text and one complete Go program. The harness builds and runs your program, computes the expected output itself, and rejects a transmission that breaks a rule below, telling you exactly why.",
		kindRules[k.Or()],
		programRules,
	}
	if k != levels.Decode {
		parts = append(parts, regionRules)
	}
	parts = append(parts, textRules, storyRules[d], "Respond with the JSON object only.")
	return strings.Join(parts, "\n\n")
}

func orderMessage(o Order) string {
	var sb strings.Builder
	sb.WriteString("Mission order\n")
	fmt.Fprintf(&sb, "Topic: %s (%s)\n", o.Topic.Title, o.Topic.Concepts)
	fmt.Fprintf(&sb, "Idiom: %s\n", o.Beat.Brief)
	fmt.Fprintf(&sb, "Kind: %s\n", o.Kind.Or())
	fmt.Fprintf(&sb, "Difficulty: %s\n", o.Difficulty)
	fmt.Fprintf(&sb, "Station system: %s (%s)\n", o.System.Name, o.System.Brief)
	fmt.Fprintf(&sb, "GOSCII's voice: %s\n", ai.Voice(o.Repaired))
	if o.Topic.Guidance != "" {
		fmt.Fprintf(&sb, "Topic rules: %s\n", o.Topic.Guidance)
	}
	if rules := ruleList(o.Beat.Requires); rules != "" {
		where := "in the astronaut's region"
		if o.Kind == levels.Decode {
			where = "in the program"
		}
		fmt.Fprintf(&sb, "The checker requires %s: %s\n", where, rules)
	}
	if rules := ruleList(o.Beat.Forbids); rules != "" {
		fmt.Fprintf(&sb, "The checker forbids: %s\n", rules)
	}
	if o.Dossier != "" {
		sb.WriteString(o.Dossier + "\n")
	}
	if o.Brief != "" {
		sb.WriteString(o.Brief + "\n")
	}
	if o.Carry != "" {
		sb.WriteString("\nIncident mode: the astronaut's program from the previous mission is the scaffold of this one. " +
			"Keep its code as it is, place the region where the new work goes, and extend it to the next stage of the same system. " +
			"The previous program:\n" + o.Carry + "\n")
	}
	if o.Wish != "" {
		fmt.Fprintf(&sb, "\nThe astronaut asked for: %q. Honour it within the topic and idiom.\n", o.Wish)
	}
	return sb.String()
}

func ruleList(rules []string) string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = engine.DescribeRule(r)
	}
	return strings.Join(out, ", ")
}

func repairMessage(d *Defect) string {
	detail := d.Detail
	if len(detail) > 2000 {
		detail = detail[:2000] + "\n[truncated]"
	}
	msg := "Transmission rejected: " + d.Rule + "."
	if detail != "" {
		msg += "\n" + detail
	}
	return msg + "\nFix it and send the complete JSON object again."
}

func str() map[string]any { return map[string]any{"type": "string"} }

func strList() map[string]any { return map[string]any{"type": "array", "items": str()} }

// schema returns the structured-output schema for a kind. Every property is
// required and no others are allowed, as strict modes demand.
func schema(k levels.Kind) map[string]any {
	props := map[string]any{
		"title":    str(),
		"story":    str(),
		"hints":    strList(),
		"solution": str(),
	}
	switch k.Or() {
	case levels.Repair:
		props["alt_inputs"] = strList()
	case levels.Fault:
		props["alt_inputs"] = strList()
		props["faulty"] = str()
		props["fault"] = str()
	case levels.Refit:
		props["alt_inputs"] = strList()
		props["seed"] = str()
	}
	required := make([]string, 0, len(props))
	for _, name := range []string{"title", "story", "hints", "solution", "alt_inputs", "faulty", "fault", "seed"} {
		if _, ok := props[name]; ok {
			required = append(required, name)
		}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}
