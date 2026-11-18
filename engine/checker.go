// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"fmt"
	"strings"
)

type CheckRule struct {
	StdoutNonempty bool   `yaml:"stdout_nonempty" json:"stdout_nonempty,omitempty"`
	StdoutContains string `yaml:"stdout_contains"  json:"stdout_contains,omitempty"`
	StdoutEquals   string `yaml:"stdout_equals"    json:"stdout_equals,omitempty"`
	// Cases replay the program on other input regions; each expected stdout
	// was computed by running the reference answer.
	Cases []Case `yaml:"cases" json:"cases,omitempty"`
	// Requires and Forbids are construct rules on the player's region (see rules.go).
	Requires []string `yaml:"requires" json:"requires,omitempty"`
	Forbids  []string `yaml:"forbids"  json:"forbids,omitempty"`
	// Prediction marks a decode mission: the player types the expected output
	// instead of code, and StdoutEquals holds the truth from a real run.
	Prediction bool `yaml:"prediction" json:"prediction,omitempty"`
}

// Case is one alternate input set with its expected output.
type Case struct {
	Input  string `yaml:"input"  json:"input"`
	Stdout string `yaml:"stdout" json:"stdout"`
}

func (c CheckRule) Verify(r RunResult) bool {
	if !r.ExitOK {
		return false
	}
	if c.StdoutNonempty && strings.TrimSpace(r.Stdout) == "" {
		return false
	}
	if c.StdoutContains != "" && !strings.Contains(r.Stdout, c.StdoutContains) {
		return false
	}
	if c.StdoutEquals != "" && !SameOutput(r.Stdout, c.StdoutEquals) {
		return false
	}
	return true
}

// SameOutput compares outputs ignoring trailing whitespace on each line and
// trailing blank lines.
func SameOutput(a, b string) bool { return normOutput(a) == normOutput(b) }

func normOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}

// Verdict is the outcome of judging the player's code.
type Verdict struct {
	Passed bool
	Result RunResult // the primary run, shown in the status pane
	// Failure explains a failed check in GOSCII's voice; empty when the run
	// itself already shows the problem (compile error, panic).
	Failure string
}

// Judge builds and runs the player's code against every part of the check.
func Judge(template, playerCode string, check CheckRule) Verdict {
	if check.Prediction {
		return judgePrediction(playerCode, check)
	}
	prog, res := Build(template, playerCode)
	if prog == nil {
		return Verdict{Result: res}
	}
	res = prog.Run()
	prog.Close()
	v := Verdict{Result: res}
	if !res.ExitOK {
		return v
	}
	if !check.Verify(res) {
		v.Failure = fmt.Sprintf("got: %q", strings.TrimSpace(res.Stdout))
		return v
	}
	if broken, err := RuleViolations(InjectCode(template, playerCode), check.Requires, check.Forbids); err == nil && len(broken) > 0 {
		v.Failure = "output matches, but the repair is off-protocol: " + strings.Join(broken, "; ")
		return v
	}
	for i, c := range check.Cases {
		got := RunCode(WithInput(template, c.Input), playerCode)
		if !got.ExitOK || !SameOutput(got.Stdout, c.Stdout) {
			v.Failure = fmt.Sprintf("telemetry shift %d/%d: the repair holds on the sample but breaks on other readings. Do not hardcode the answer.", i+1, len(check.Cases))
			return v
		}
	}
	v.Passed = true
	return v
}

func judgePrediction(prediction string, check CheckRule) Verdict {
	res := RunResult{Stdout: prediction, ExitOK: true, Compiled: true}
	v := Verdict{Result: res}
	if strings.TrimSpace(prediction) == "" {
		v.Failure = "no prediction transmitted"
		return v
	}
	if SameOutput(prediction, check.StdoutEquals) {
		v.Passed = true
		return v
	}
	v.Failure = "prediction does not match the recovered log"
	return v
}
