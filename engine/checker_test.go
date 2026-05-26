// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import "testing"

func TestVerifyFailsOnNonZeroExit(t *testing.T) {
	c := CheckRule{StdoutEquals: "42"}
	if c.Verify(RunResult{Stdout: "42", ExitOK: false}) {
		t.Fatal("expected failure when exit is non-zero")
	}
}

func TestVerifyStdoutEquals(t *testing.T) {
	c := CheckRule{StdoutEquals: "42"}

	if !c.Verify(RunResult{Stdout: "42\n", ExitOK: true}) {
		t.Fatal("expected pass: trailing newline should be trimmed")
	}
	if c.Verify(RunResult{Stdout: "43\n", ExitOK: true}) {
		t.Fatal("expected fail: wrong value")
	}
}

func TestVerifyStdoutContains(t *testing.T) {
	c := CheckRule{StdoutContains: "ok"}

	if !c.Verify(RunResult{Stdout: "status: ok\n", ExitOK: true}) {
		t.Fatal("expected pass: output contains substring")
	}
	if c.Verify(RunResult{Stdout: "status: fail\n", ExitOK: true}) {
		t.Fatal("expected fail: substring absent")
	}
}

func TestVerifyStdoutNonempty(t *testing.T) {
	c := CheckRule{StdoutNonempty: true}

	if !c.Verify(RunResult{Stdout: "anything\n", ExitOK: true}) {
		t.Fatal("expected pass: non-empty output")
	}
	if c.Verify(RunResult{Stdout: "   \n", ExitOK: true}) {
		t.Fatal("expected fail: whitespace-only output")
	}
}

func TestVerifyEmptyRulePassesOnExit(t *testing.T) {
	c := CheckRule{}
	if !c.Verify(RunResult{ExitOK: true}) {
		t.Fatal("expected pass: no assertions, exit ok")
	}
}
