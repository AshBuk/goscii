// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"strconv"
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

func TestTemplateHeaderReturnsLinesAboveMarker(t *testing.T) {
	got := TemplateHeader(baseTemplate)
	want := "package main\n\nimport \"fmt\"\n\nfunc main() {"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTemplateHeaderNoMarker(t *testing.T) {
	if got := TemplateHeader("package main\nfunc main() {}"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestTemplateCodeOffsetPointsToFirstPlayerLine(t *testing.T) {
	offset := TemplateCodeOffset(baseTemplate)
	injected := InjectCode(baseTemplate, "MARKER")
	lines := strings.Split(injected, "\n")
	// lines are 0-indexed; offset is 1-based line number
	if lines[offset-1] != "MARKER" {
		t.Fatalf("offset %d points to %q, want \"MARKER\"\nfull file:\n%s",
			offset, lines[offset-1], injected)
	}
}

func TestNormalizeErrorsRemapsLineNumbers(t *testing.T) {
	offset := TemplateCodeOffset(baseTemplate)
	raw := "# command-line-arguments\n/tmp/goscii_123.go:" + strconv.Itoa(offset) + ":5: undefined: x"
	got := NormalizeErrors(raw, offset, 3)
	want := "line 1:5: undefined: x"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeErrorsOutOfRangeLabelsAsGOSCII(t *testing.T) {
	// Generated line 1 is in the template header, not the player's code.
	got := NormalizeErrors("/tmp/goscii_abc.go:1:1: undefined: name", 7, 3)
	if !strings.HasPrefix(got, "GOSCII line ") {
		t.Fatalf("expected GOSCII: prefix for out-of-range line, got %q", got)
	}
}

func TestNormalizeErrorsScaffoldLineAfterPlayerCode(t *testing.T) {
	offset := TemplateCodeOffset(baseTemplate) // 7
	// Player has 2 lines; scaffold starts at generated line 9.
	scaffoldLine := strconv.Itoa(offset + 2)
	raw := "/tmp/goscii_abc.go:" + scaffoldLine + ":3: undefined: x"
	got := NormalizeErrors(raw, offset, 2)
	if !strings.HasPrefix(got, "GOSCII line ") {
		t.Fatalf("expected GOSCII: prefix, got %q", got)
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
