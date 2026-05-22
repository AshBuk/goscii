// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type RunResult struct {
	Stdout string
	Stderr string
	ExitOK bool
}

const (
	codeStart = "// === YOUR CODE HERE ==="
	codeEnd   = "// === END ==="
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
	f.Close()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", f.Name())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	return RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		ExitOK: err == nil,
	}
}
