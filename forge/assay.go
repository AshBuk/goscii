// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package forge

import (
	"encoding/json"
	"fmt"
	"go/format"
	"math/rand/v2"
	"regexp"
	"strings"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

const (
	codeStart = "// === YOUR CODE HERE ==="
	codeEnd   = "// === END ==="
	maxCases  = 3
	maxLines  = 8 // decode output ceiling
)

// draft mirrors the structured output the signal returns.
type draft struct {
	Title     string   `json:"title"`
	Story     string   `json:"story"`
	Hints     []string `json:"hints"`
	Solution  string   `json:"solution"`
	AltInputs []string `json:"alt_inputs"`
	Faulty    string   `json:"faulty"`
	Fault     string   `json:"fault"`
	Seed      string   `json:"seed"`
}

// Defect is a rule a draft broke. The signal receives it verbatim as repair feedback.
type Defect struct {
	Rule   string
	Detail string
}

func (d *Defect) Error() string {
	if d.Detail == "" {
		return d.Rule
	}
	return d.Rule + ": " + d.Detail
}

func defect(rule, detail string) *Defect { return &Defect{Rule: rule, Detail: detail} }

// assayed is a draft that passed every check, with the truth computed by runs.
type assayed struct {
	mission  *levels.Mission
	template string
}

// assay parses a reply and runs every check for the order's kind.
func assay(raw string, o Order) (*assayed, *Defect) {
	d, def := parseDraft(raw)
	if def != nil {
		return nil, def
	}
	if def := checkText(d); def != nil {
		return nil, def
	}
	if def := checkSource(d.Solution); def != nil {
		return nil, def
	}
	switch o.Kind.Or() {
	case levels.Decode:
		return assayDecode(d, o)
	default:
		return assayRegion(d, o)
	}
}

func parseDraft(raw string) (draft, *Defect) {
	var d draft
	if err := json.Unmarshal([]byte(stripFences(raw)), &d); err != nil {
		return d, defect("reply is not the JSON object", err.Error())
	}
	return d, nil
}

func checkText(d draft) *Defect {
	switch {
	case strings.TrimSpace(d.Title) == "":
		return defect("missing title", "")
	case strings.TrimSpace(d.Story) == "":
		return defect("missing story", "")
	case len(d.Hints) == 0:
		return defect("missing hints", "")
	case strings.TrimSpace(d.Solution) == "":
		return defect("missing solution", "")
	}
	return nil
}

func checkSource(src string) *Defect {
	if bad := engine.ForbiddenImport(src); bad != "" {
		return defect("forbidden import", fmt.Sprintf("%q is blocked aboard; use another package", bad))
	}
	if _, err := format.Source([]byte(src)); err != nil {
		return defect("solution does not parse as Go", err.Error())
	}
	return nil
}

// assayRegion covers repair, fault and refit: kinds with an editable region.
func assayRegion(d draft, o Order) (*assayed, *Defect) {
	if strings.Count(d.Solution, codeStart) != 1 || strings.Count(d.Solution, codeEnd) != 1 {
		return nil, defect("region markers missing or repeated", "use '"+codeStart+"' and '"+codeEnd+"' exactly once each")
	}
	start, end := strings.Index(d.Solution, codeStart), strings.Index(d.Solution, codeEnd)
	if end < start || !strings.Contains(d.Solution[start:end], "\n") {
		return nil, defect("region markers out of order", "put '"+codeStart+"' on its own line above '"+codeEnd+"'")
	}
	region, _ := engine.PlayerRegion(d.Solution)
	answer := dedent(region)
	if strings.TrimSpace(answer) == "" {
		return nil, defect("empty region", "the code between the markers is the reference answer and cannot be empty")
	}
	tmpl := hollow(d.Solution)
	if !engine.TemplateWellFormed(tmpl) {
		return nil, defect("scaffold does not parse without the region", "something opened above "+codeStart+" closes inside it; move the markers so the scaffold is balanced")
	}

	out, def := reference(tmpl, answer)
	if def != nil {
		return nil, def
	}
	if broken, err := engine.RuleViolations(engine.InjectCode(tmpl, answer), o.Beat.Requires, o.Beat.Forbids); err != nil {
		return nil, defect("solution does not parse", err.Error())
	} else if len(broken) > 0 {
		return nil, defect("reference breaks the idiom rule", strings.Join(broken, "; ")+". The region must use the idiom of the order")
	}
	if hollow := engine.RunCode(tmpl, ""); hollow.ExitOK && engine.SameOutput(hollow.Stdout, out) {
		return nil, defect("scaffold prints the answer by itself", "with the region empty the program still prints the target output; move the output into the region")
	}

	m := baseMission(d, o)
	m.Answer = answer
	m.Check.StdoutEquals = out
	m.Check.Requires = o.Beat.Requires
	m.Check.Forbids = o.Beat.Forbids
	m.Check.Cases = cases(tmpl, answer, out, d.AltInputs)
	if def := seedKind(m, tmpl, d, out, o); def != nil {
		return nil, def
	}
	if def := checkSight(m, d.Solution, tmpl, out); def != nil {
		return nil, def
	}
	return &assayed{mission: m, template: tmpl}, nil
}

// seedKind runs the extra checks of kinds that start the editor with code.
func seedKind(m *levels.Mission, tmpl string, d draft, out string, o Order) *Defect {
	switch o.Kind.Or() {
	case levels.Fault:
		if def := checkFault(tmpl, d, out); def != nil {
			return def
		}
		m.Seed = dedent(d.Faulty)
		m.Fault = strings.TrimSpace(d.Fault)
	case levels.Refit:
		if def := checkRefit(tmpl, d, out, o); def != nil {
			return def
		}
		m.Seed = dedent(d.Seed)
	}
	return nil
}

// reference builds the reference once and runs it three times: it must exit
// cleanly, print something, and print the same thing every time.
func reference(tmpl, answer string) (string, *Defect) {
	prog, res := engine.Build(tmpl, answer)
	if prog == nil {
		return "", defect("reference does not compile", res.Stderr)
	}
	defer prog.Close()
	var first string
	for i := range 3 {
		r := prog.Run()
		if !r.ExitOK {
			return "", defect("reference fails at runtime", r.Stderr)
		}
		if i == 0 {
			first = r.Stdout
			continue
		}
		if r.Stdout != first {
			return "", defect("output is not deterministic", fmt.Sprintf("run 1 printed:\n%s\nrun %d printed:\n%s", first, i+1, r.Stdout))
		}
	}
	if strings.TrimSpace(first) == "" {
		return "", defect("reference prints nothing", "the astronaut needs output to check against")
	}
	return strings.TrimRight(first, "\n"), nil
}

// cases keeps the alternate inputs that build, run, and change the output.
func cases(tmpl, answer, out string, alts []string) []engine.Case {
	if !engine.HasInput(tmpl) {
		return nil
	}
	var cs []engine.Case
	for _, in := range alts {
		if len(cs) == maxCases || strings.TrimSpace(in) == "" {
			continue
		}
		r := engine.RunCode(engine.WithInput(tmpl, in), answer)
		if !r.ExitOK || strings.TrimSpace(r.Stdout) == "" || engine.SameOutput(r.Stdout, out) {
			continue
		}
		cs = append(cs, engine.Case{Input: strings.Trim(in, "\n"), Stdout: strings.TrimRight(r.Stdout, "\n")})
	}
	return cs
}

func checkFault(tmpl string, d draft, out string) *Defect {
	faulty := dedent(d.Faulty)
	switch {
	case strings.TrimSpace(faulty) == "":
		return defect("missing faulty", "send the region with the planted fault")
	case strings.TrimSpace(d.Fault) == "":
		return defect("missing fault", "name the planted fault in one sentence")
	}
	prog, res := engine.Build(tmpl, faulty)
	if prog == nil {
		return defect("faulty code does not compile", res.Stderr+"\nPlant a runtime or logic fault, not a compile error")
	}
	defer prog.Close()
	r := prog.Run()
	if r.ExitOK && engine.SameOutput(r.Stdout, out) {
		return defect("planted fault is harmless", "the faulty region prints the correct output; the fault must cause a panic, a deadlock or wrong output")
	}
	return nil
}

func checkRefit(tmpl string, d draft, out string, o Order) *Defect {
	seed := dedent(d.Seed)
	if strings.TrimSpace(seed) == "" {
		return defect("missing seed", "send the clumsy working version of the region")
	}
	r := engine.RunCode(tmpl, seed)
	if !r.ExitOK || !engine.SameOutput(r.Stdout, out) {
		return defect("seed does not work", "the seed must already print exactly the target output. It printed:\n"+r.Stdout+r.Stderr)
	}
	broken, err := engine.RuleViolations(engine.InjectCode(tmpl, seed), o.Beat.Requires, o.Beat.Forbids)
	if err != nil {
		return defect("seed does not parse", err.Error())
	}
	if len(broken) == 0 {
		return defect("seed already follows protocol", "write the seed without the idiom so there is something to refit")
	}
	return nil
}

func assayDecode(d draft, o Order) (*assayed, *Defect) {
	if strings.Contains(d.Solution, codeStart) {
		return nil, defect("decode program has region markers", "a decode program is read, not edited; remove the markers")
	}
	out, def := reference(d.Solution, "")
	if def != nil {
		return nil, def
	}
	if n := strings.Count(out, "\n") + 1; n > maxLines {
		return nil, defect("output too long to predict", fmt.Sprintf("it prints %d lines; keep it to %d", n, maxLines))
	}
	if broken, err := engine.RuleViolations(d.Solution, o.Beat.Requires, nil); err == nil && len(broken) > 0 {
		return nil, defect("program skips the idiom", strings.Join(broken, "; "))
	}
	m := baseMission(d, o)
	m.Answer = out
	m.Check = engine.CheckRule{StdoutEquals: out, Prediction: true}
	return &assayed{mission: m, template: d.Solution}, nil
}

var outputLine = regexp.MustCompile(`(?im)\n*^\s*output\s*:.*$`)

func baseMission(d draft, o Order) *levels.Mission {
	hints := make([]string, 0, len(d.Hints))
	for _, h := range d.Hints {
		if h = strings.TrimSpace(h); h != "" {
			hints = append(hints, h)
		}
	}
	return &levels.Mission{
		ID:         missionID(o),
		Title:      strings.TrimSpace(d.Title),
		Concept:    o.Topic.Slug,
		Beat:       o.Beat.Slug,
		Kind:       o.Kind.Or(),
		Difficulty: o.Difficulty,
		System:     o.System.Slug,
		Story:      strings.TrimSpace(outputLine.ReplaceAllString(d.Story, "")),
		Hints:      hints,
	}
}

// hollow empties the region of a solution, keeping the end marker's indentation.
func hollow(solution string) string {
	start := strings.Index(solution, codeStart)
	end := strings.Index(solution, codeEnd)
	lineStart := strings.LastIndex(solution[:end], "\n") + 1
	return solution[:start+len(codeStart)] + "\n" + solution[lineStart:]
}

func missionID(o Order) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet))]
	}
	return fmt.Sprintf("%s-%s-%s", o.Topic.Slug, o.Difficulty, b)
}

// dedent strips blank edge lines and the indentation common to all lines.
func dedent(s string) string {
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	common := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if common == -1 || n < common {
			common = n
		}
	}
	for i, l := range lines {
		if len(l) >= common && common > 0 {
			lines[i] = l[common:]
		} else if strings.TrimSpace(l) == "" {
			lines[i] = ""
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), " \t\n")
}

// stripFences removes markdown code fences some endpoints add despite json mode.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.Index(s, "\n"); i > 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "```"); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
