// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Formatter strips gofmt indentation from AI-generated code before it reaches the cockpit.
package tui

import (
	"go/format"
	"strings"
)

// formatGoSnippet formats the code snippet if it is syntactically valid Go.
// It first tries the snippet as a func main() body (statements/expressions),
// then as package-level declarations (types, methods, funcs). Returns ok=false
// for prose.
func formatGoSnippet(code string) (string, bool) {
	if out, ok := formatMainBody(code); ok {
		return out, true
	}
	return formatFile(code)
}

// formatMainBody formats code that belongs inside func main().
func formatMainBody(code string) (string, bool) {
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

// formatFile formats package-level code (types, methods, func main and friends).
func formatFile(code string) (string, bool) {
	out, err := format.Source([]byte("package main\n\n" + code + "\n"))
	if err != nil {
		return "", false
	}
	body := strings.TrimPrefix(string(out), "package main\n")
	body = strings.Trim(body, "\n")
	if body == "" {
		return "", false
	}
	return strings.ReplaceAll(body, "\t", "    "), true
}
