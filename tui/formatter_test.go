// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"
	"testing"
)

func TestFmtGoSnippetFormatsValidCode(t *testing.T) {
	code := `if x == 1 { fmt.Println("yes") } else { fmt.Println("no") }`
	got, ok := formatGoSnippet(code)
	if !ok {
		t.Fatal("expected ok=true for valid Go snippet")
	}
	if !strings.Contains(got, "\n") {
		t.Fatalf("expected multi-line output, got %q", got)
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("expected no tabs in output (should use 4 spaces), got %q", got)
	}
}

func TestFmtGoSnippetNestedIndentUsesSpaces(t *testing.T) {
	code := "if true {\nfmt.Println(1)\n}"
	got, ok := formatGoSnippet(code)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("expected 4-space indent, got tabs: %q", got)
	}
	if !strings.Contains(got, "    fmt.Println(1)") {
		t.Fatalf("expected 4-space nested indent, got %q", got)
	}
}

func TestFmtGoSnippetReturnsFalseForProse(t *testing.T) {
	prose := "Two transmission protocols detected. Standard: `var name = value`"
	_, ok := formatGoSnippet(prose)
	if ok {
		t.Fatal("expected ok=false for prose text")
	}
}

func TestFmtGoSnippetSingleStatement(t *testing.T) {
	got, ok := formatGoSnippet(`x := 42`)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "x := 42" {
		t.Fatalf("got %q", got)
	}
}

func TestFmtGoSnippetPackageLevelDeclarations(t *testing.T) {
	code := "type Sensor struct{ Zone string }\nfunc (s Sensor) Status() string { return s.Zone }\nfunc main() { fmt.Println(Sensor{}.Status()) }"
	got, ok := formatGoSnippet(code)
	if !ok {
		t.Fatal("expected ok=true for package-level declarations")
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("expected 4-space indent, got tabs: %q", got)
	}
	if !strings.Contains(got, "type Sensor struct") || !strings.Contains(got, "func (s Sensor) Status()") {
		t.Fatalf("expected declarations preserved, got %q", got)
	}
}
