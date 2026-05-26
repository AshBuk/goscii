// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"strings"
	"testing"
)

const baseTemplate = `package main

import "fmt"

func main() {
	// === YOUR CODE HERE ===
	// === END ===
	fmt.Println(x)
}`

func TestScaffoldAfterExtractsPostEndCode(t *testing.T) {
	got := ScaffoldAfter(baseTemplate)
	if got != "fmt.Println(x)" {
		t.Fatalf("got %q", got)
	}
}

func TestInjectCodePlacesPlayerCodeBetweenMarkers(t *testing.T) {
	got := InjectCode(baseTemplate, "x := 42")
	startIdx := strings.Index(got, codeStart)
	playerIdx := strings.Index(got, "x := 42")
	endIdx := strings.Index(got, codeEnd)
	scaffoldIdx := strings.Index(got, "fmt.Println(x)")

	if startIdx == -1 || playerIdx == -1 || endIdx == -1 || scaffoldIdx == -1 {
		t.Fatalf("injected code is missing expected sections:\n%s", got)
	}
	if startIdx >= playerIdx || playerIdx >= endIdx || endIdx >= scaffoldIdx {
		t.Fatalf("player code was not injected into the editable section:\n%s", got)
	}
}

func TestRunCodeExecutesInjectedProgram(t *testing.T) {
	got := RunCode(baseTemplate, "x := 42")
	if !got.ExitOK {
		t.Fatalf("expected successful run, stderr: %s", got.Stderr)
	}
	if got.Stdout != "42\n" {
		t.Fatalf("expected stdout %q, got %q", "42\n", got.Stdout)
	}
}

func TestRunCodeReportsCompileFailure(t *testing.T) {
	got := RunCode(baseTemplate, "fmt.Println(missing)")
	if got.ExitOK {
		t.Fatal("expected compile failure")
	}
	if strings.TrimSpace(got.Stderr) == "" {
		t.Fatal("expected compiler stderr")
	}
}

func TestScaffoldAfterNoMarker(t *testing.T) {
	if got := ScaffoldAfter("package main\nfunc main() {}"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestScaffoldAfterEmptyBody(t *testing.T) {
	tmpl := `package main

func main() {
	// === YOUR CODE HERE ===
	// === END ===
}`
	if got := ScaffoldAfter(tmpl); got != "" {
		t.Fatalf("expected empty scaffold, got %q", got)
	}
}
