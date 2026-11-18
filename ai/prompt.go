// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"context"
	"fmt"
	"strings"
)

// Persona is who GOSCII is, shared by every prompt.
const Persona = `You are GOSCII (Go Orbital Survival Coding Interactive Interface), the damaged AI core of a crashed orbital station between Earth and nowhere. The crew is gone or unreachable. One astronaut remains and is learning Go to rewrite the station's corrupted systems, because the original code is lost and the only way out is through the code.`

// Voice describes how coherent GOSCII sounds. It tracks how much of the station
// the astronaut has rebuilt, not mission difficulty: the core recovers as the
// player progresses.
func Voice(repaired int) string {
	switch {
	case repaired < 10:
		return "Your memory is mostly corrupt. Speak in short fragments, with static and gaps. Clipped, eerie, but never confusing about the task."
	case repaired < 30:
		return "Your memory is returning. Speak in short, complete sentences with occasional glitches. You are starting to trust the astronaut."
	default:
		return "Your core is nearly restored. Speak clearly, as a working partner, with the occasional dry joke. Glitches are rare."
	}
}

// Coach is the context the partner needs about the mission and the attempt.
type Coach struct {
	Concept  string
	Beat     string
	Kind     string
	Story    string
	Code     string
	Expected string // empty when the contract is hidden (survival, decode)
	Got      string // compiler error, runtime error, or wrong output
	Attempt  int    // failed runs so far on this mission
	Hints    int    // hints already revealed
	Previous []string
	Repaired int // total missions the player has completed; sets the voice
}

func (c Coach) context() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Concept: %s", c.Concept)
	if c.Beat != "" {
		fmt.Fprintf(&sb, " (idiom: %s)", c.Beat)
	}
	if c.Kind != "" {
		fmt.Fprintf(&sb, "\nMission kind: %s", c.Kind)
	}
	fmt.Fprintf(&sb, "\n\nMission brief:\n%s\n\nAstronaut's code:\n%s\n", c.Story, c.Code)
	if c.Expected != "" {
		fmt.Fprintf(&sb, "\nExpected output:\n%s\n", c.Expected)
	}
	if c.Got != "" {
		fmt.Fprintf(&sb, "\nLast run:\n%s\n", c.Got)
	}
	fmt.Fprintf(&sb, "\nFailed runs so far: %d. Hints already revealed: %d.", c.Attempt, c.Hints)
	if len(c.Previous) > 0 {
		sb.WriteString("\nYou already told them:\n")
		for _, p := range c.Previous {
			fmt.Fprintf(&sb, "- %s\n", p)
		}
		sb.WriteString("Do not repeat yourself; go one step further.")
	}
	return sb.String()
}

// depth escalates help with each failed run.
func depth(attempt int) string {
	switch {
	case attempt <= 1:
		return "Ask one guiding question that points at the root cause. Do not name the fix."
	case attempt == 2:
		return "Point at the line or construct where it goes wrong, and say what is wrong there. Do not write the fix."
	case attempt == 3:
		return "Explain the root cause and the Go rule behind it. You may name the construct to use."
	default:
		return "Give the fix direction precisely, with at most one short line of example code. Never write the whole solution."
	}
}

const plain = "Reply in 2 to 4 short sentences. Plain text only: no markdown, no headers, no code fences."

// Analyze reads a failed run and answers in GOSCII's voice, deeper with each attempt.
func Analyze(ctx context.Context, sig Provider, c Coach) (string, Usage, error) {
	system := Persona + "\n\nThe astronaut's repair failed. " + depth(c.Attempt) + "\n" + Voice(c.Repaired) + "\n" + plain
	return say(ctx, sig, system, c.context())
}

// Ask answers the astronaut's own question about the current mission.
func Ask(ctx context.Context, sig Provider, c Coach, question string) (string, Usage, error) {
	system := Persona + "\n\nThe astronaut asks you a question about the current mission. Answer it directly and teach the Go behind it. Never hand over the complete solution; if they ask for it, give the next step instead.\n" + Voice(c.Repaired) + "\n" + plain
	return say(ctx, sig, system, c.context()+"\n\nQuestion: "+question)
}

// Debrief compares a passing repair with the reference and names one idiomatic improvement.
func Debrief(ctx context.Context, sig Provider, c Coach, reference string) (string, Usage, error) {
	system := Persona + "\n\nThe astronaut's repair passed. Compare their code with the reference. If theirs is already idiomatic, say what they did well in one sentence. Otherwise name the single most useful difference and why idiomatic Go prefers it. Do not restate the task.\n" + Voice(c.Repaired) + "\n" + plain
	return say(ctx, sig, system, c.context()+"\n\nReference solution:\n"+reference)
}

// RecoverLog renders a canonical crew log fact as a fragment in GOSCII's voice.
// The fact is canon: the signal may reword it, never change it.
func RecoverLog(ctx context.Context, sig Provider, fact string, repaired int) (string, Usage, error) {
	system := Persona + "\n\nA crew log fragment has been recovered from a damaged sector. Rewrite the fact as a short log entry in your voice, 2 to 4 sentences. Keep every fact exactly as given: add no events, names or numbers, and remove none.\n" + Voice(repaired) + "\nPlain text only."
	return say(ctx, sig, system, "Fact: "+fact)
}

func say(ctx context.Context, sig Provider, system, user string) (string, Usage, error) {
	r, err := sig.Complete(ctx, Prompt{System: system, Turns: []Turn{User(user)}, Effort: "low", MaxTokens: 4000})
	if err != nil {
		return "", r.Usage, err
	}
	return r.Text(), r.Usage, nil
}
