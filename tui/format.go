// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"go/format"
	"strings"
)

// formatGoSnippet tries to format code as a func main() body using go/format.
// Returns (formatted, true) on success. Returns ("", false) if the code is
// not syntactically valid Go — e.g. prose answers from handcrafted levels.
func formatGoSnippet(code string) (string, bool) {
	src := "package main\nfunc main() {\n" + code + "\n}\n"
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", false
	}
	body := string(out)
	start := strings.Index(body, "{\n")
	end := strings.LastIndex(body, "\n}")
	if start == -1 || end == -1 || start >= end {
		return "", false
	}
	// Strip one leading tab per line (gofmt indents the func body).
	lines := strings.Split(body[start+2:end], "\n")
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, "\t")
	}
	return strings.Join(lines, "\n"), true
}
