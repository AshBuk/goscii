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

func TestTemplateFooterReturnsPostEndCodeVerbatim(t *testing.T) {
	got := TemplateFooter(baseTemplate)
	want := "\tfmt.Println(x)\n}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
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

func TestTemplateFooterNoMarker(t *testing.T) {
	if got := TemplateFooter("package main\nfunc main() {}"); got != "" {
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

func TestTemplateFooterClosingBraceOnly(t *testing.T) {
	tmpl := `package main

func main() {
	// === YOUR CODE HERE ===
	// === END ===
}`
	if got := TemplateFooter(tmpl); got != "}" {
		t.Fatalf("expected closing brace, got %q", got)
	}
}

func TestTemplateWellFormedAcceptsFuncMainHole(t *testing.T) {
	if !TemplateWellFormed(baseTemplate) {
		t.Fatal("func main() template should be well-formed")
	}
}

func TestTemplateWellFormedAcceptsPackageLevelHole(t *testing.T) {
	tmpl := `package main

import "fmt"

// === YOUR CODE HERE ===
// === END ===`
	if !TemplateWellFormed(tmpl) {
		t.Fatal("package-level template should be well-formed")
	}
}

func TestTemplateWellFormedRejectsOpenImportGroup(t *testing.T) {
	// Marker placed inside an unclosed import group — the scaffold cannot parse.
	tmpl := `package main

import (
	"fmt"
	"sort"
	// === YOUR CODE HERE ===
	// === END ===
}`
	if TemplateWellFormed(tmpl) {
		t.Fatal("template with a split import group should be rejected")
	}
}

func TestTemplateWellFormedRejectsMissingMarker(t *testing.T) {
	if TemplateWellFormed("package main\nfunc main() {}") {
		t.Fatal("template without markers should be rejected")
	}
}

func TestTemplateFooterPackageLevelMarkers(t *testing.T) {
	tmpl := `package main

import "fmt"

// === YOUR CODE HERE ===
// === END ===`
	if got := TemplateFooter(tmpl); got != "" {
		t.Fatalf("expected empty footer for package-level markers, got %q", got)
	}
}

func TestForbiddenImport(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "clean stdlib",
			code: "package main\nimport \"fmt\"\nfunc main() { fmt.Println(1) }",
			want: "",
		},
		{
			name: "os and net are allowed",
			code: "package main\nimport (\n\t\"os\"\n\t\"net/http\"\n)\nvar _ = os.Args\nvar _ = http.Get",
			want: "",
		},
		{
			name: "os/exec blocked",
			code: "package main\nimport \"os/exec\"\nvar _ = exec.Command",
			want: "os/exec",
		},
		{
			name: "blank-imported syscall blocked",
			code: "package main\nimport _ \"syscall\"\nfunc main() {}",
			want: "syscall",
		},
		{
			name: "x/sys prefix blocked",
			code: "package main\nimport \"golang.org/x/sys/unix\"\nvar _ = unix.Exit",
			want: "golang.org/x/sys/unix",
		},
		{
			name: "parse error yields empty (compiler will reject)",
			code: "this is not go",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForbiddenImport(tt.code); got != tt.want {
				t.Fatalf("ForbiddenImport = %q, want %q", got, tt.want)
			}
		})
	}
}
