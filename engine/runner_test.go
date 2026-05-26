// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import "testing"

const baseTemplate = `package main

import "fmt"

func main() {
	// === YOUR CODE HERE ===
	// === END ===
	fmt.Println(x)
}`

func TestScaffoldAfterExtractsPostEndCode(t *testing.T) {
	got := ScaffoldAfter(baseTemplate)
	if got != "fmt.Println(x)" {
		t.Fatalf("got %q", got)
	}
}

func TestScaffoldAfterNoMarker(t *testing.T) {
	if got := ScaffoldAfter("package main\nfunc main() {}"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestScaffoldAfterEmptyBody(t *testing.T) {
	tmpl := `package main

func main() {
	// === YOUR CODE HERE ===
	// === END ===
}`
	if got := ScaffoldAfter(tmpl); got != "" {
		t.Fatalf("expected empty scaffold, got %q", got)
	}
}
