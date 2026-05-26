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
	if strings.HasPrefix(got, "\t") {
		t.Fatalf("expected leading tab stripped, got %q", got)
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
