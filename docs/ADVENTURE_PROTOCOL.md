# Adventure Protocol

*How to author a bundled offline adventure for GOSCII.*

An **adventure** is a named, ordered track of missions that ships inside the binary
and runs without a "Signal" (API key). The onboarding campaign is one. This document is the contract a new adventure must follow so the engine and cockpit can load, render,
run, and check it.

A bundled adventure may be **handcrafted or AI-generated** with human in the loop — both are welcome.

---

## 1. Layout

```
levels/adventures/<adventure-name>/
  01_<slug>/
    level.yaml
    template.txt
  02_<slug>/
    level.yaml
    template.txt
  ...
```

- One directory per mission, `NN_<slug>` with a **two-digit zero-padded** prefix.
- Missions load in `embed.FS` order, which is **lexical** — so `01_…`, `02_…`, `10_…`
  sort correctly. Do not skip or duplicate numbers.
- Everything is embedded at build time via `levels/FS`; no runtime file access.

## 2. `level.yaml`

```yaml
id: 01_variables_1        # unique, stable; used as the progress key
title: "Cold Boot"        # shown in the cockpit header
concept: variables        # MUST equal a file in assets/art/ (without .txt)
difficulty: easy          # easy | medium | hard | survival
story: |                  # GOSCII's in-world framing (the WORLD panel)
  ...
hints:                    # shown one at a time on ctrl+h
  - "conceptual nudge"
  - "concrete syntax — never the full answer"
answer: |                 # the exact code that goes between the markers
  name := "Ava"
check:
  stdout_equals: "..."    # see §4
```

- **`concept`** selects the world art. It must match a filename in `assets/art/`
  (e.g. `variables` → `assets/art/variables.txt`). Reusing a slug across missions is
  fine — they share the art. Add new art there if you need a new concept.
- **`difficulty`** tunes the cockpit: on `survival` the signal contract, hints, and
  answer are hidden. Keep teaching levels `easy`/`medium` so the task stays clear.
- **`answer`** is the gofmt-formatted Go that belongs *between the markers*. The
  cockpit reveals it on `ctrl+a`. It may also be prose for an open beat (rare).
- **`story`** carries the narrative voice; the *task* itself must stay crisp and end
  with the expected `Output:`. Use the project's sci-fi voice — see
  [AGENTS.md](../AGENTS.md).

## 3. `template.txt`

The scaffold the player edits. Two markers delimit the editable region:

```go
package main

import "fmt"

func main() {
	raw := 215
	// === YOUR CODE HERE ===

	// === END ===
	fmt.Println(celsius)
}
```

- Lines **above** `// === YOUR CODE HERE ===` (header) and **below** `// === END ===`
  (footer) are read-only scaffold shown around the editor. Header and footer must
  parse as valid Go on their own.
- The player's code is injected between the markers, then compiled and run.
- Two shapes:
  - **compute-and-print** — markers inside `main`; the player computes *and* prints.
  - **compute-only** — the footer ends with `fmt.Println(result)` referencing a value
    the player must define. This pins the output format and forces the lesson (as
    above, `fmt.Println(celsius)` forces declaring `celsius`). Prefer it when there is
    one clear result.
- Package-level work (functions, types, methods, interfaces) goes with the markers at
  package level and a `main()` in the footer that exercises it.
- Blocked imports (`os/exec`, `syscall`, `unsafe`, `plugin`, `golang.org/x/sys`) are
  rejected for safety — generated code runs automatically during checking.

## 4. Checks

`check` decides pass/fail against the program's stdout (exit must be clean):

| field | meaning |
|-------|---------|
| `stdout_equals`   | trimmed stdout must equal this (preferred — deterministic, teaches exact output) |
| `stdout_contains` | stdout must contain this substring |
| `stdout_nonempty` | any non-empty stdout passes (reserve for open beats, e.g. "declare your name") |

Prefer `stdout_equals`. Concurrency missions (goroutines, channels) must produce
**deterministic** output — sort or count before printing, or the check can't hold.

## 5. Register and validate

1. Add the track to the registry in [levels/loader.go](../levels/loader.go):

   ```go
   const MyTrack Adventure = "my-track"
   var BundledAdventures = []Adventure{Onboarding, MyTrack}
   ```

2. Build and test, then play it end to end:

   ```sh
   go build ./... && go vet ./... && go test ./...
   goscii adventure my-track
   ```

   Solve every mission against its own `check`; reveal each `answer` (`ctrl+a`) and
   confirm it satisfies the check too.

## 6. Authoring conventions

- One new idiom per mission; carry a small but genuine algorithm, not just syntax.
- Two hints: one conceptual, one concrete-syntax — never the full answer.
- Keep the narrator's voice in the `story`; keep the task unambiguous.
- Map each mission to an in-world beat so the track reads as one story, not a worksheet.
