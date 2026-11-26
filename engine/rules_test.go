// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"strings"
	"testing"
)

func region(code string) string {
	return "package main\n\nimport \"fmt\"\n\n// === YOUR CODE HERE ===\n" + code + "\n// === END ===\n"
}

func TestConstructsSeeOnlyThePlayerRegion(t *testing.T) {
	prog := "package main\n\nfunc helper() { defer func() {}() }\n\n// === YOUR CODE HERE ===\nfunc main() {}\n// === END ===\n"
	found, err := Constructs(prog)
	if err != nil {
		t.Fatal(err)
	}
	if found.has("defer") {
		t.Fatal("defer outside the region must not count")
	}
}

func TestConstructRules(t *testing.T) {
	cases := []struct {
		rule string
		code string
		want bool
	}{
		{"select", "func main() { c := make(chan int); select { case <-c: } }", true},
		{"select-default", "func main() { c := make(chan int); select { case <-c: default: } }", true},
		{"select-default", "func main() { c := make(chan int); select { case <-c: } }", false},
		{"chan-buffered", "func main() { _ = make(chan int, 3) }", true},
		{"chan-buffered", "func main() { _ = make(chan int) }", false},
		{"pointer-method", "type M struct{ n int }\nfunc (m *M) Up() { m.n++ }\nfunc main() {}", true},
		{"pointer-method", "type M struct{ n int }\nfunc (m M) Up() {}\nfunc main() {}", false},
		{"generics", "func Max[T int | float64](a, b T) T { return a }\nfunc main() {}", true},
		{"constraint", "func Id[T any](a T) T { return a }\nfunc main() {}", false},
		{"recursion", "func f(n int) int { if n == 0 { return 0 }; return f(n - 1) }\nfunc main() {}", true},
		{"closure", "func gen() func() int { n := 0; return func() int { n++; return n } }\nfunc main() {}", true},
		{"comma-ok", "func main() { m := map[string]int{}; v, ok := m[\"a\"]; fmt.Println(v, ok) }", true},
		{"call:errors.Is", "func main() { _ = errors.Is(nil, nil) }", true},
		{"call:.Go", "func main() { var wg sync.WaitGroup; wg.Go(func() {}); wg.Wait() }", true},
		{"sync.Mutex", "var mu sync.Mutex\nfunc main() {}", true},
		{"slices.*", "func main() { slices.Sort([]int{}) }", true},
		{"sort", "func main() { slices.Sort([]int{}) }", true},
		{".Done", "func main() { var ctx context.Context; <-ctx.Done() }", true},
		{"conversion", "func main() { x := 3; fmt.Println(float64(x)) }", true},
		{"yield-check", "func s(yield func(int) bool) { if !yield(1) { return } }\nfunc main() {}", true},
		{"slice-3", "func main() { s := []int{1, 2, 3}; _ = s[0:1:1] }", true},
		{"embed", "type A struct{}\ntype B struct{ A }\nfunc main() {}", true},
	}
	for _, c := range cases {
		found, err := Constructs(region(c.code))
		if err != nil {
			t.Fatalf("%s: %v", c.rule, err)
		}
		if got := found.has(c.rule); got != c.want {
			t.Errorf("%s on %q: got %v, want %v", c.rule, c.code, got, c.want)
		}
	}
}

func TestValidRule(t *testing.T) {
	for _, r := range []string{"select", "call:append", "call:errors.Is", "sync.Mutex", "slices.*", ".Done", "call:.Go"} {
		if !ValidRule(r) {
			t.Errorf("%q should be valid", r)
		}
	}
	if ValidRule("selec") {
		t.Error("a typo must not be a valid rule")
	}
}

const inputTemplate = `package main

import "fmt"

func main() {
	// === INPUT ===
	a, b := 2, 3
	// === END INPUT ===
	// === YOUR CODE HERE ===
	// === END ===
}`

func TestWithInputKeepsLineCount(t *testing.T) {
	got := WithInput(inputTemplate, "a, b := 10, 20")
	if strings.Count(got, "\n") != strings.Count(inputTemplate, "\n") {
		t.Fatalf("line count changed:\n%s", got)
	}
	if !strings.Contains(got, "a, b := 10, 20") || strings.Contains(got, "2, 3") {
		t.Fatalf("input not swapped:\n%s", got)
	}
	if TemplateCodeOffset(got) != TemplateCodeOffset(inputTemplate) {
		t.Fatal("swapping input must not move the player region")
	}
}

func TestDisplayScaffoldHidesInputMarkers(t *testing.T) {
	got := DisplayScaffold(TemplateHeader(inputTemplate))
	if strings.Contains(got, "INPUT") {
		t.Fatalf("input markers shown:\n%s", got)
	}
}

func TestJudgeRunsCases(t *testing.T) {
	check := CheckRule{
		StdoutEquals: "5",
		Cases:        []Case{{Input: "a, b := 10, 20", Stdout: "30"}},
	}
	if v := Judge(inputTemplate, "fmt.Println(a + b)", check); !v.Passed {
		t.Fatalf("general solution must pass: %+v", v)
	}
	v := Judge(inputTemplate, "fmt.Println(5)\n_, _ = a, b", check)
	if v.Passed || !strings.Contains(v.Failure, "telemetry shift") || !strings.Contains(v.Failure, "a, b := 10, 20") || !strings.Contains(v.Failure, `got: "5"`) {
		t.Fatalf("hardcoded solution must fail the shifted case: %+v", v)
	}
}

func TestJudgeChecksRules(t *testing.T) {
	check := CheckRule{StdoutEquals: "5", Requires: []string{"switch"}}
	v := Judge(inputTemplate, "fmt.Println(a + b)", check)
	if v.Passed || !strings.Contains(v.Failure, "off-protocol") {
		t.Fatalf("missing construct must fail: %+v", v)
	}
}

func TestRunHidesHostEnvironment(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-secret")
	tmpl := "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\t// === YOUR CODE HERE ===\n\t// === END ===\n}"
	res := RunCode(tmpl, `fmt.Print(os.Getenv("ANTHROPIC_API_KEY"))`)
	if !res.ExitOK {
		t.Fatal(res.Stderr)
	}
	if res.Stdout != "" {
		t.Fatalf("mission code read a host secret: %q", res.Stdout)
	}
}

func TestBuildErrorIsMarkedUncompiled(t *testing.T) {
	res := RunCode(inputTemplate, "fmt.Println(c)")
	if res.Compiled || !strings.Contains(res.Stderr, "line 1:") {
		t.Fatalf("want compiler error mapped to editor line 1, got %+v", res)
	}
}
