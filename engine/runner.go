// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package engine executes player code in isolated orbit and verifies the results.
package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RunResult struct {
	Stdout   string
	Stderr   string
	ExitOK   bool
	Compiled bool // false when the build failed; Stderr then holds compiler output
	TimedOut bool
}

const (
	codeStart  = "// === YOUR CODE HERE ==="
	codeEnd    = "// === END ==="
	inputStart = "// === INPUT ==="
	inputEnd   = "// === END INPUT ==="
	buildLimit = 60 * time.Second
	runLimit   = 5 * time.Second
	sourceName = "goscii_main.go" // NormalizeErrors matches the goscii_ prefix
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

// PlayerRegion returns the code between the player markers of a full program.
func PlayerRegion(program string) (string, bool) {
	_, rest, ok := strings.Cut(program, codeStart)
	if !ok {
		return "", false
	}
	region, _, ok := strings.Cut(rest, codeEnd)
	if !ok {
		return "", false
	}
	return region, true
}

// HasInput reports whether the template marks an input region for alternate cases.
func HasInput(template string) bool {
	_, ok := InputRegion(template)
	return ok
}

// InputRegion returns the code between the input markers.
func InputRegion(template string) (string, bool) {
	_, rest, ok := strings.Cut(template, inputStart)
	if !ok {
		return "", false
	}
	region, _, ok := strings.Cut(rest, inputEnd)
	return region, ok
}

// WithInput swaps the input region of a template for another set of values.
// The line count of the region is kept so compiler line mapping still holds.
func WithInput(template, input string) string {
	startIdx := strings.Index(template, inputStart)
	endIdx := strings.Index(template, inputEnd)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return template
	}
	before := template[:startIdx+len(inputStart)]
	after := template[endIdx:]
	old := template[startIdx+len(inputStart) : endIdx]
	body := "\n" + strings.Trim(input, "\n") + "\n"
	if pad := strings.Count(old, "\n") - strings.Count(body, "\n"); pad > 0 {
		body += strings.Repeat("\n", pad)
	}
	return before + body + after
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

// DisplayScaffold drops the input marker lines from scaffold text shown to the player.
func DisplayScaffold(s string) string {
	lines := strings.Split(s, "\n")
	out := lines[:0]
	for _, l := range lines {
		if t := strings.TrimSpace(l); t == inputStart || t == inputEnd {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
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

// TemplateFooter returns the lines below the player's section — any
// mission-wired code after // === END === plus the closing braces — for
// read-only display beneath the editor. Indentation is preserved verbatim so
// the displayed structure matches the generated file. Returns "" when there is
// no end marker or nothing follows it.
func TemplateFooter(template string) string {
	_, rest, ok := strings.Cut(template, codeEnd)
	if !ok {
		return ""
	}
	return strings.Trim(rest, "\n")
}

// TemplateWellFormed reports whether the scaffold — everything outside the
// player's editable region (header + footer) — parses as valid Go on its own.
func TemplateWellFormed(template string) bool {
	header := TemplateHeader(template)
	if header == "" {
		return false
	}
	skeleton := header + "\n" + TemplateFooter(template)
	_, err := parser.ParseFile(token.NewFileSet(), "", skeleton, parser.SkipObjectResolution)
	return err == nil
}

// errLineRe matches the temp-file prefix in go build error output, e.g.:
//
//	./goscii_main.go:8:5:
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

// forbiddenImports lists packages blocked for safety. Generated code runs
// automatically during validation, so it must not be able to execute shell
// commands or make raw syscalls. os and net are intentionally allowed — the
// files and http topics use them; isolating those is the Docker sandbox's job.
var forbiddenImports = map[string]bool{
	"os/exec": true,
	"syscall": true,
	"unsafe":  true,
	"plugin":  true,
}

// ForbiddenImport reports the first disallowed import path in code, or "" if none.
// A parse error yields "" - the compiler will reject the code on its own, and code
// that does not compile never runs.
func ForbiddenImport(code string) string {
	f, err := parser.ParseFile(token.NewFileSet(), "", code, parser.ImportsOnly)
	if err != nil {
		return ""
	}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		if forbiddenImports[p] || strings.HasPrefix(p, "golang.org/x/sys") {
			return p
		}
	}
	return ""
}

// Program is a compiled mission binary in its own throwaway directory.
type Program struct {
	dir string
	bin string
}

// Close removes the binary and everything the program wrote.
func (p *Program) Close() { _ = os.RemoveAll(p.dir) }

// Build compiles template with playerCode injected. On failure the returned
// RunResult carries the normalized compiler output and the Program is nil.
func Build(template, playerCode string) (*Program, RunResult) {
	code := InjectCode(template, playerCode)
	if bad := ForbiddenImport(code); bad != "" {
		return nil, RunResult{Stderr: fmt.Sprintf("blocked import %q: not allowed for safety reasons", bad)}
	}
	dir, err := os.MkdirTemp("", "goscii_run_")
	if err != nil {
		return nil, RunResult{Stderr: err.Error()}
	}
	src := filepath.Join(dir, sourceName)
	if err := os.WriteFile(src, []byte(code), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, RunResult{Stderr: err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildLimit)
	defer cancel()
	bin := filepath.Join(dir, "mission")
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, sourceName)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GO111MODULE=on", "GOWORK=off")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		msg := stderr.String()
		if ctx.Err() == context.DeadlineExceeded {
			msg = "mission build timed out"
		}
		lines := strings.Count(playerCode, "\n") + 1
		return nil, RunResult{Stderr: NormalizeErrors(msg, TemplateCodeOffset(template), lines)}
	}
	return &Program{dir: dir, bin: bin}, RunResult{Compiled: true}
}

// Run executes the program with a clean environment: no inherited variables,
// so API keys in the player's shell never reach mission code, and HOME and
// TMPDIR point into the throwaway directory.
func (p *Program) Run() RunResult {
	ctx, cancel := context.WithTimeout(context.Background(), runLimit)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, p.bin)
	cmd.Dir = p.dir
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + p.dir,
		"TMPDIR=" + p.dir,
		"LANG=C.UTF-8",
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := RunResult{Stdout: stdout.String(), Compiled: true, ExitOK: err == nil}
	res.Stderr = strings.TrimSpace(stripTracePaths(stderr.String(), p.dir))
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
		res.ExitOK = false
		res.Stderr = strings.TrimSpace(res.Stderr + "\nmission timed out")
	}
	return res
}

// stripTracePaths shortens panic traces that point into the run directory.
func stripTracePaths(s, dir string) string {
	return strings.ReplaceAll(s, dir+string(filepath.Separator), "")
}

// RunCode builds and runs once.
func RunCode(template, playerCode string) RunResult {
	prog, res := Build(template, playerCode)
	if prog == nil {
		return res
	}
	defer prog.Close()
	return prog.Run()
}
