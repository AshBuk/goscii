// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

const (
	inputStart = "// === INPUT ==="
	inputEnd   = "// === END INPUT ==="
)

// plainNumbers need no telling: counters, halves, parity.
var plainNumbers = map[float64]bool{0: true, 1: true, 2: true}

// valueWords are printed by Go values themselves; the astronaut needs no telling.
var valueWords = map[string]bool{"true": true, "false": true, "nil": true}

var numberRe = regexp.MustCompile(`\d+(?:\.\d+)?`)

// checkSight holds the mission to what the astronaut can see: data the answer
// reads is shifted, and every number and word in the answer's literals appears
// in the story, the scaffold, the seed or the sample output (hidden in survival).
func checkSight(m *levels.Mission, solution, tmpl, out string) *Defect {
	if def := checkInputs(solution); def != nil {
		return def
	}
	if engine.HasInput(tmpl) && len(m.Check.Cases) == 0 {
		return defect("no alternate input changes the output",
			"every alt_inputs body must build, run and print something other than the sample output, so a hardcoded answer fails")
	}
	seen := m.Story + "\n" + tmpl + "\n" + m.Seed
	if m.Difficulty != levels.Survival {
		seen += "\n" + out
	}
	return checkFacts(engine.InjectCode(tmpl, m.Answer), seen)
}

// source is a parsed program with the line span of its astronaut region.
type source struct {
	fset   *token.FileSet
	file   *ast.File
	region span
}

func parse(src string) (source, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return source{}, false // the reference run already reported it
	}
	return source{fset, file, markerLines(src, codeStart, codeEnd)}, true
}

func (s source) line(n ast.Node) int { return s.fset.Position(n.Pos()).Line }

func (s source) inRegion(n ast.Node) bool { return s.region.holds(s.line(n)) }

// checkInputs rejects scaffold variables with literal values that the region
// reads from outside the input block: no shift could ever reach them.
func checkInputs(solution string) *Defect {
	s, ok := parse(solution)
	if !ok {
		return nil
	}
	data := s.literalVars(markerLines(solution, inputStart, inputEnd))
	var read []string
	ast.Inspect(s.file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && data[id.Name] && s.inRegion(id) && !slices.Contains(read, id.Name) {
			read = append(read, id.Name)
		}
		return true
	})
	if len(read) == 0 {
		return nil
	}
	return defect("input data outside the input block",
		fmt.Sprintf("the region reads %s, declared with literal values outside %q and %q. Move them into the input block and send alt_inputs that change the output", strings.Join(read, ", "), inputStart, inputEnd))
}

// literalVars names the variables declared with literal values outside both
// the region and the input block.
func (s source) literalVars(input span) map[string]bool {
	data := map[string]bool{}
	declare := func(names []ast.Expr, values []ast.Expr) {
		if len(names) != len(values) {
			return
		}
		for i, e := range names {
			id, ok := e.(*ast.Ident)
			if ok && !s.inRegion(id) && !input.holds(s.line(id)) && literal(values[i]) {
				data[id.Name] = true
			}
		}
	}
	ast.Inspect(s.file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				declare(n.Lhs, n.Rhs)
			}
		case *ast.GenDecl:
			if n.Tok != token.VAR {
				return true
			}
			for _, spec := range n.Specs {
				v := spec.(*ast.ValueSpec)
				names := make([]ast.Expr, len(v.Names))
				for i, id := range v.Names {
					names[i] = id
				}
				declare(names, v.Values)
			}
		}
		return true
	})
	return data
}

// literal reports whether e is built from literal values only.
func literal(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return e.Name == "true" || e.Name == "false"
	case *ast.UnaryExpr:
		return literal(e.X)
	case *ast.ParenExpr:
		return literal(e.X)
	case *ast.KeyValueExpr:
		return literal(e.Value)
	case *ast.CompositeLit:
		for _, el := range e.Elts {
			if !literal(el) {
				return false
			}
		}
		return true
	}
	return false
}

// checkFacts rejects numbers and words in the region's literals that the
// astronaut cannot see: a hidden threshold or message makes the task a guess.
func checkFacts(program, seen string) *Defect {
	s, ok := parse(program)
	if !ok {
		return nil
	}
	tuning := s.tuningArgs()
	numbers, known := numberSet(seen), words(seen)
	var hidden []string
	ast.Inspect(s.file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || !s.inRegion(lit) {
			return true
		}
		switch lit.Kind {
		case token.INT, token.FLOAT:
			if v, ok := number(lit.Value); ok && !tuning[lit] && !plainNumbers[v] && !numbers[v] {
				hidden = append(hidden, lit.Value)
			}
		case token.STRING:
			text, _ := strconv.Unquote(lit.Value)
			for w := range words(text) {
				if !known[w] && !valueWords[w] {
					hidden = append(hidden, strconv.Quote(w))
				}
			}
		}
		return true
	})
	if len(hidden) == 0 {
		return nil
	}
	slices.Sort(hidden)
	return defect("the answer depends on facts the astronaut never sees",
		fmt.Sprintf("the reference region uses %s, which appear nowhere the astronaut can read. The astronaut sees the story, the scaffold and the sample output, never the alternate inputs. Name each threshold and message in the story, or declare it in the scaffold", strings.Join(slices.Compact(hidden), ", ")))
}

// tuningArgs marks arguments passed straight to a library call or make:
// strconv bases, float precision and capacities are tuning, not facts.
func (s source) tuningArgs() map[ast.Expr]bool {
	pkgs := map[string]bool{"make": true}
	for _, im := range s.file.Imports {
		path, _ := strconv.Unquote(im.Path.Value)
		name := path[strings.LastIndex(path, "/")+1:]
		if im.Name != nil {
			name = im.Name.Name
		}
		pkgs[name] = true
	}
	tuning := map[ast.Expr]bool{}
	ast.Inspect(s.file, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && pkgs[callee(c)] {
			for _, a := range c.Args {
				tuning[a] = true
			}
		}
		return true
	})
	return tuning
}

// callee names the package or builtin of a call: "strconv" for strconv.Itoa.
func callee(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// words returns the lowercase letter runs of s, two letters or longer.
func words(s string) map[string]bool {
	ws := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if len([]rune(w)) > 1 {
			ws[w] = true
		}
	}
	return ws
}

func numberSet(s string) map[float64]bool {
	ns := map[float64]bool{}
	for _, t := range numberRe.FindAllString(s, -1) {
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			ns[v] = true
		}
	}
	return ns
}

// number reads a Go numeric literal: 0x1F, 1_000, 2.5.
func number(lit string) (float64, bool) {
	if i, err := strconv.ParseInt(lit, 0, 64); err == nil {
		return float64(i), true
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(lit, "_", ""), 64)
	return v, err == nil
}

// span is a marker pair's line range, exclusive; zero when absent.
type span struct{ lo, hi int }

func (s span) holds(line int) bool { return line > s.lo && line < s.hi }

func markerLines(src, start, end string) span {
	var s span
	for i, l := range strings.Split(src, "\n") {
		switch strings.TrimSpace(l) {
		case start:
			if s.lo == 0 {
				s.lo = i + 1
			}
		case end:
			if s.hi == 0 {
				s.hi = i + 1
			}
		}
	}
	if s.hi <= s.lo {
		return span{}
	}
	return s
}
