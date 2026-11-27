// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

// Construct rules name Go constructs the player's region must or must not use.
// Forms:
//
//	select          a keyword construct from the vocabulary below
//	sync.Mutex      any use of a package selector
//	slices.*        any selector on a package
//	.Done           any field or method selector with that name
//	call:append     a call to a builtin or plain function
//	call:errors.Is  a call to a package function
//	call:.Go        a method call with that name
var constructs = map[string]string{
	"short-var":       "a := declaration",
	"var":             "a var declaration",
	"const":           "a const declaration",
	"iota":            "iota",
	"conversion":      "an explicit type conversion",
	"multi-assign":    "a multiple assignment",
	"func":            "a function declaration",
	"multi-return":    "a function with several results",
	"named-returns":   "named result parameters",
	"variadic":        "a variadic parameter",
	"func-param":      "a function-typed parameter",
	"closure":         "a returned closure",
	"recursion":       "a recursive function",
	"defer":           "defer",
	"if":              "if",
	"if-init":         "an if whose init statement computes a value",
	"if-err":          "an error check against nil",
	"switch":          "an expression switch",
	"switch-tagless":  "a tagless switch",
	"for":             "a for loop",
	"for-classic":     "a three-clause for loop",
	"for-cond":        "a condition-only for loop",
	"range":           "a range loop",
	"label":           "a labeled statement",
	"slice-expr":      "a slice expression",
	"slice-3":         "a three-index slice expression",
	"nested-slice":    "a slice of slices",
	"map":             "a map",
	"comma-ok":        "a comma-ok read",
	"inc":             "an increment",
	"sort":            "sorting",
	"empty-struct":    "struct{}",
	"struct":          "a struct type",
	"method":          "a method",
	"pointer-method":  "a pointer-receiver method",
	"embed":           "an embedded field",
	"pointer":         "a pointer",
	"eq":              "an == comparison",
	"nil":             "nil",
	"interface":       "an interface type",
	"interface-embed": "an embedded interface",
	"type-assert":     "a type assertion",
	"type-switch":     "a type switch",
	"go":              "a go statement",
	"chan":            "a channel operation",
	"chan-buffered":   "a buffered channel",
	"chan-dir":        "a directional channel type",
	"select":          "select",
	"select-default":  "select with a default case",
	"struct-tag":      "a struct tag",
	"generics":        "type parameters",
	"generic-type":    "a generic type declaration",
	"constraint":      "a type constraint other than any",
	"yield-check":     "a check of yield's result",
}

// DescribeRule returns a short human phrase for a rule.
func DescribeRule(rule string) string {
	if d, ok := constructs[rule]; ok {
		return d
	}
	return strings.TrimPrefix(rule, "call:")
}

// ValidRule reports whether a rule is understood by the checker.
func ValidRule(rule string) bool {
	if _, ok := constructs[rule]; ok {
		return true
	}
	name := strings.TrimPrefix(rule, "call:")
	return name != "" && (strings.Contains(name, ".") || rule != name)
}

// RuleViolations lists the rules the player's region breaks: requires not
// found, forbids found. program is the full source with the player markers.
func RuleViolations(program string, requires, forbids []string) ([]string, error) {
	if len(requires) == 0 && len(forbids) == 0 {
		return nil, nil
	}
	found, err := Constructs(program)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range requires {
		if !found.has(r) {
			out = append(out, "missing "+DescribeRule(r))
		}
	}
	for _, r := range forbids {
		if found.has(r) {
			out = append(out, "must not use "+DescribeRule(r))
		}
	}
	return out, nil
}

// Found is the set of constructs present in a player region.
type Found struct {
	kinds     map[string]bool
	selectors map[string]bool // "pkg.Name" and ".Name"
	calls     map[string]bool // "fn", "pkg.Name", ".Name"
}

func (f Found) has(rule string) bool {
	if name, ok := strings.CutPrefix(rule, "call:"); ok {
		return f.calls[name]
	}
	if pkg, ok := strings.CutSuffix(rule, ".*"); ok {
		for s := range f.selectors {
			if strings.HasPrefix(s, pkg+".") {
				return true
			}
		}
		return false
	}
	if strings.Contains(rule, ".") {
		return f.selectors[rule]
	}
	return f.kinds[rule]
}

// Constructs parses program and collects constructs whose position lies inside
// the player's region, or anywhere in the file when it has no player markers.
func Constructs(program string) (Found, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, sourceName, program, parser.SkipObjectResolution)
	if err != nil {
		return Found{}, fmt.Errorf("parse: %w", err)
	}
	lo, hi := regionLines(program)
	if lo == 0 {
		hi = strings.Count(program, "\n") + 2
	}
	f := Found{kinds: map[string]bool{}, selectors: map[string]bool{}, calls: map[string]bool{}}
	in := func(n ast.Node) bool {
		l := fset.Position(n.Pos()).Line
		return l > lo && l < hi
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || !in(n) {
			return true
		}
		f.visit(n)
		return true
	})
	return f, nil
}

// regionLines returns the line numbers of the start and end markers.
func regionLines(program string) (lo, hi int) {
	for i, l := range strings.Split(program, "\n") {
		switch strings.TrimSpace(l) {
		case codeStart:
			if lo == 0 {
				lo = i + 1
			}
		case codeEnd:
			if hi == 0 {
				hi = i + 1
			}
		}
	}
	if hi <= lo {
		return 0, 0
	}
	return lo, hi
}

var basicTypes = []string{
	"bool", "string", "int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"byte", "rune", "float32", "float64", "complex64", "complex128",
}

func (f Found) visit(n ast.Node) { //nolint:gocyclo // one case per construct
	k := f.kinds
	switch n := n.(type) {
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			k["short-var"] = true
		}
		if len(n.Lhs) >= 2 && len(n.Rhs) == len(n.Lhs) {
			k["multi-assign"] = true
		}
		if len(n.Lhs) == 2 && len(n.Rhs) == 1 {
			switch r := n.Rhs[0].(type) {
			case *ast.IndexExpr, *ast.TypeAssertExpr:
				k["comma-ok"] = true
			case *ast.UnaryExpr:
				if r.Op == token.ARROW {
					k["comma-ok"] = true
				}
			}
		}
	case *ast.ValueSpec:
		if len(n.Values) == 2 && len(n.Names) == 2 {
			if _, ok := n.Values[0].(*ast.IndexExpr); ok {
				k["comma-ok"] = true
			}
		}
	case *ast.GenDecl:
		switch n.Tok {
		case token.VAR:
			k["var"] = true
		case token.CONST:
			k["const"] = true
		}
	case *ast.Ident:
		switch n.Name {
		case "iota":
			k["iota"] = true
		case "nil":
			k["nil"] = true
		}
	case *ast.FuncDecl:
		f.visitFuncDecl(n)
	case *ast.FuncType:
		f.visitFuncType(n)
	case *ast.ReturnStmt:
		for _, r := range n.Results {
			if _, ok := r.(*ast.FuncLit); ok {
				k["closure"] = true
			}
		}
	case *ast.DeferStmt:
		k["defer"] = true
	case *ast.GoStmt:
		k["go"] = true
	case *ast.IfStmt:
		k["if"] = true
		if computes(n.Init) {
			k["if-init"] = true
		}
		if u, ok := n.Cond.(*ast.UnaryExpr); ok && u.Op == token.NOT {
			if c, ok := u.X.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "yield" {
					k["yield-check"] = true
				}
			}
		}
	case *ast.BinaryExpr:
		switch n.Op {
		case token.EQL:
			k["eq"] = true
		case token.NEQ:
			if id, ok := n.Y.(*ast.Ident); ok && id.Name == "nil" {
				k["if-err"] = true
			}
		}
	case *ast.SwitchStmt:
		if n.Tag != nil {
			k["switch"] = true
		} else {
			k["switch-tagless"] = true
		}
	case *ast.TypeSwitchStmt:
		k["type-switch"] = true
	case *ast.ForStmt:
		k["for"] = true
		switch {
		case n.Init != nil && n.Post != nil:
			k["for-classic"] = true
		case n.Cond != nil && n.Init == nil && n.Post == nil:
			k["for-cond"] = true
		}
	case *ast.RangeStmt:
		k["for"] = true
		k["range"] = true
	case *ast.LabeledStmt:
		k["label"] = true
	case *ast.SliceExpr:
		k["slice-expr"] = true
		if n.Slice3 {
			k["slice-3"] = true
		}
	case *ast.ArrayType:
		if inner, ok := n.Elt.(*ast.ArrayType); ok && n.Len == nil && inner.Len == nil {
			k["nested-slice"] = true
		}
	case *ast.MapType:
		k["map"] = true
	case *ast.IncDecStmt:
		if n.Tok == token.INC {
			k["inc"] = true
		}
	case *ast.StructType:
		if n.Fields == nil || len(n.Fields.List) == 0 {
			k["empty-struct"] = true
			break
		}
		k["struct"] = true
		for _, fld := range n.Fields.List {
			if len(fld.Names) == 0 {
				k["embed"] = true
			}
			if fld.Tag != nil {
				k["struct-tag"] = true
			}
		}
	case *ast.InterfaceType:
		k["interface"] = true
		for _, m := range n.Methods.List {
			if len(m.Names) == 0 {
				k["interface-embed"] = true
			}
		}
	case *ast.StarExpr:
		k["pointer"] = true
	case *ast.UnaryExpr:
		switch n.Op {
		case token.AND:
			k["pointer"] = true
		case token.ARROW:
			k["chan"] = true
		}
	case *ast.SendStmt:
		k["chan"] = true
	case *ast.ChanType:
		k["chan"] = true
		if n.Dir != ast.SEND|ast.RECV {
			k["chan-dir"] = true
		}
	case *ast.SelectStmt:
		k["select"] = true
		for _, s := range n.Body.List {
			if cc, ok := s.(*ast.CommClause); ok && cc.Comm == nil {
				k["select-default"] = true
			}
		}
	case *ast.TypeAssertExpr:
		if n.Type != nil {
			k["type-assert"] = true
		}
	case *ast.TypeSpec:
		if n.TypeParams != nil && len(n.TypeParams.List) > 0 {
			k["generics"] = true
			k["generic-type"] = true
			f.visitTypeParams(n.TypeParams)
		}
	case *ast.SelectorExpr:
		name := "." + n.Sel.Name
		f.selectors[name] = true
		if id, ok := n.X.(*ast.Ident); ok {
			f.selectors[id.Name+name] = true
			if id.Name == "sort" || (id.Name == "slices" && strings.HasPrefix(n.Sel.Name, "Sort")) {
				k["sort"] = true
			}
		}
	case *ast.CallExpr:
		f.visitCall(n)
	}
}

// computes reports whether an if's init assigns something worked out: a call,
// a lookup, an expression. Copying a name or a literal teaches nothing.
func computes(init ast.Stmt) bool {
	a, ok := init.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, e := range a.Rhs {
		for {
			p, ok := e.(*ast.ParenExpr)
			if !ok {
				break
			}
			e = p.X
		}
		switch e.(type) {
		case *ast.Ident, *ast.BasicLit:
		default:
			return true
		}
	}
	return false
}

func (f Found) visitFuncDecl(n *ast.FuncDecl) {
	k := f.kinds
	if n.Recv != nil && len(n.Recv.List) > 0 {
		k["method"] = true
		if _, ok := n.Recv.List[0].Type.(*ast.StarExpr); ok {
			k["pointer-method"] = true
		}
	} else if n.Name.Name != "main" && n.Name.Name != "init" {
		k["func"] = true
	}
	if n.Type.TypeParams != nil && len(n.Type.TypeParams.List) > 0 {
		k["generics"] = true
		f.visitTypeParams(n.Type.TypeParams)
	}
	if recursive(n) {
		k["recursion"] = true
	}
}

// recursive reports whether a function or method calls itself by name.
func recursive(n *ast.FuncDecl) bool {
	if n.Body == nil {
		return false
	}
	self, found := n.Name.Name, false
	ast.Inspect(n.Body, func(m ast.Node) bool {
		c, ok := m.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		switch fn := c.Fun.(type) {
		case *ast.Ident:
			found = fn.Name == self && n.Recv == nil
		case *ast.SelectorExpr:
			found = fn.Sel.Name == self && n.Recv != nil
		}
		return !found
	})
	return found
}

func (f Found) visitFuncType(n *ast.FuncType) {
	k := f.kinds
	if n.Results != nil {
		count := 0
		for _, r := range n.Results.List {
			if len(r.Names) > 0 {
				k["named-returns"] = true
			}
			count += max(1, len(r.Names))
		}
		if count >= 2 {
			k["multi-return"] = true
		}
	}
	if n.Params == nil {
		return
	}
	for _, p := range n.Params.List {
		switch p.Type.(type) {
		case *ast.Ellipsis:
			k["variadic"] = true
		case *ast.FuncType:
			k["func-param"] = true
		}
	}
}

func (f Found) visitTypeParams(list *ast.FieldList) {
	for _, p := range list.List {
		if id, ok := p.Type.(*ast.Ident); ok && id.Name == "any" {
			continue
		}
		f.kinds["constraint"] = true
	}
}

func (f Found) visitCall(n *ast.CallExpr) {
	switch fn := n.Fun.(type) {
	case *ast.Ident:
		f.calls[fn.Name] = true
		if slices.Contains(basicTypes, fn.Name) && len(n.Args) == 1 {
			f.kinds["conversion"] = true
		}
		if fn.Name == "make" && len(n.Args) >= 2 {
			if _, ok := n.Args[0].(*ast.ChanType); ok {
				f.kinds["chan-buffered"] = true
			}
		}
	case *ast.SelectorExpr:
		f.calls["."+fn.Sel.Name] = true
		if id, ok := fn.X.(*ast.Ident); ok {
			f.calls[id.Name+"."+fn.Sel.Name] = true
		}
	}
}
