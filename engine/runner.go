// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package engine executes player code in isolated orbit and verifies the results.
package engine

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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

func RunCode(template, playerCode string) RunResult {
	code := InjectCode(template, playerCode)

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
		Stderr: stderr.String(),
		ExitOK: err == nil,
	}
}
