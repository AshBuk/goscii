// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package engine executes player code in isolated orbit and verifies the results.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RunResult struct {
	Stdout string
	Stderr string
	ExitOK bool
}

const (
	codeStart = "// === YOUR CODE HERE ==="
	codeEnd   = "// === END ==="
	runLimit  = 5 * time.Second
)

func InjectCode(template, playerCode string) string {
	startIdx := strings.Index(template, codeStart)
	endIdx := strings.Index(template, codeEnd)
	if startIdx == -1 || endIdx == -1 {
		return template
	}
	before := template[:startIdx+len(codeStart)]
	after := template[endIdx:]
	return before + "\n" + playerCode + "\n" + after
}

// TemplateHeader returns the lines above the player's section — package, imports,
// func main() { — for read-only display in the editor pane.
func TemplateHeader(template string) string {
	before, _, ok := strings.Cut(template, codeStart)
	if !ok {
		return ""
	}
	lineStart := strings.LastIndex(before, "\n")
	if lineStart == -1 {
		return ""
	}
	return strings.TrimRight(before[:lineStart], "\n")
}

// TemplateCodeOffset returns the generated-file line number where the player's
// first line lands. Used to map compiler errors back to editor line numbers:
//
//	editorLine = generatedLine - offset + 1
func TemplateCodeOffset(template string) int {
	startIdx := strings.Index(template, codeStart)
	if startIdx == -1 {
		return 0
	}
	before := template[:startIdx+len(codeStart)]
	// \n count in before = newlines up to end of marker line.
	// +1 for the injected \n that terminates the marker line,
	// +1 for 1-based line numbering.
	return strings.Count(before, "\n") + 2
}

// ScaffoldAfter returns the code that runs after the player's section -
// lines between // === END === and the closing brace of main().
func ScaffoldAfter(template string) string {
	_, rest, ok := strings.Cut(template, codeEnd)
	if !ok {
		return ""
	}
	rest = strings.TrimLeft(rest, "\n")
	if i := strings.LastIndex(rest, "}"); i != -1 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

// errLineRe matches the temp-file prefix in go run error output, e.g.:
//
//	/tmp/goscii_756850478.go:8:5:
var errLineRe = regexp.MustCompile(`[^\s:]*goscii_\w+\.go:(\d+):`)

// NormalizeErrors remaps compiler error line numbers from the generated file
// to editor line numbers (1-based from the player's first line) and strips the
// temp-file path so the output is clean for display.
//
// Lines outside [offset, offset+playerLineCount-1] are in template/scaffold
// code the player did not write; they are labelled "GOSCII:"
// (the in-world voice for mission-wired code)
func NormalizeErrors(stderr string, offset, playerLineCount int) string {
	// Drop the "# command-line-arguments" build noise header.
	out := strings.TrimPrefix(stderr, "# command-line-arguments\n")

	out = errLineRe.ReplaceAllStringFunc(out, func(match string) string {
		sub := errLineRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil {
			return match
		}
		editorLine := n - offset + 1
		if editorLine < 1 || editorLine > playerLineCount {
			return fmt.Sprintf("GOSCII line %d:", n)
		}
		return fmt.Sprintf("line %d:", editorLine)
	})

	return strings.TrimSpace(out)
}

func RunCode(template, playerCode string) RunResult {
	code := InjectCode(template, playerCode)
	offset := TemplateCodeOffset(template)
	playerLineCount := strings.Count(playerCode, "\n") + 1

	f, err := os.CreateTemp("", "goscii_*.go")
	if err != nil {
		return RunResult{Stderr: err.Error()}
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(code); err != nil {
		f.Close()
		return RunResult{Stderr: err.Error()}
	}
	if err := f.Close(); err != nil {
		return RunResult{Stderr: err.Error()}
	}

	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), runLimit)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", f.Name())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		stderr.WriteString("mission timed out")
	}

	return RunResult{
		Stdout: stdout.String(),
		Stderr: NormalizeErrors(stderr.String(), offset, playerLineCount),
		ExitOK: err == nil,
	}
}
