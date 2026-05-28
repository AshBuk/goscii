// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Formatter strips gofmt indentation from AI-generated code before it reaches the cockpit.
package tui

import (
	"go/format"
	"strings"
)

// formatGoSnippet format the code snippet if its syntactically valid Go.
func formatGoSnippet(code string) (string, bool) {
	src := "package main\nfunc main() {\n" + code + "\n}\n"
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", false
	}
	body := string(out)
	const funcDecl = "func main() {\n"
	start := strings.Index(body, funcDecl)
	end := strings.LastIndex(body, "\n}")
	if start == -1 || end == -1 || start >= end {
		return "", false
	}
	// Strip one leading tab (func body indent), then convert remaining tabs to 4 spaces.
	lines := strings.Split(body[start+len(funcDecl):end], "\n")
	for i, l := range lines {
		lines[i] = strings.ReplaceAll(strings.TrimPrefix(l, "\t"), "\t", "    ")
	}
	return strings.Join(lines, "\n"), true
}
