# Adventure Protocol

*How to author a bundled offline adventure for GOSCII.*

An **adventure** is a named, ordered track of missions that ships inside the binary
and runs without a "Signal" (API key). The onboarding campaign is one. This document is the contract a new adventure must follow so the engine and cockpit can load, render,
run, and check it.

A bundled adventure may be **handcrafted or AI-generated** with a human in the loop. Both are welcome. `goscii forge` writes a verified draft track to start from (see §5).

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
- Missions load in `embed.FS` order, which is **lexical**, so `01_…`, `02_…`, `10_…`
  sort correctly. Do not skip or duplicate numbers.
- Everything is embedded at build time via `levels/FS`; no runtime file access.

## 2. `level.yaml`

```yaml
id: 01_variables_1        # unique, stable; used as the progress key
title: "Cold Boot"        # shown in the cockpit header
concept: variables        # MUST equal a file in assets/art/ (without .txt)
difficulty: easy          # easy | medium | hard | survival
kind: repair              # repair | fault | decode | refit; empty means repair
story: |                  # GOSCII's in-world framing (the WORLD panel)
  ...
hints:                    # shown one at a time on ctrl+h
  - "conceptual nudge"
  - "concrete syntax, never the full answer"
answer: |                 # the exact code that goes between the markers
  name := "Ava"
check:
  stdout_equals: "..."    # see §4
```

Optional fields: `beat` (the idiom slug from `ai/topics.go`), `system` (a station system from `engine/station.go`, repaired when the mission passes), `seed` (initial editor content for fault and refit missions) and `fault` (one sentence naming the planted fault, read only by the analyzer).

- **`concept`** selects the world art. It must match a filename in `assets/art/`
  (e.g. `variables` → `assets/art/variables.txt`). Reusing a slug across missions is
  fine: they share the art. Add new art there if you need a new concept.
- **`difficulty`** tunes the cockpit: on `survival` the signal contract, hints, and
  answer are hidden. Keep teaching levels `easy`/`medium` so the task stays clear.
- **`answer`** is the gofmt-formatted Go that belongs *between the markers*. The
  cockpit reveals it on `ctrl+a`. For a decode mission it is the predicted output.
- **`story`** carries the narrative voice; the *task* itself must stay crisp and end
  with the expected `Output:`. Use the project's sci-fi voice, see
  [AGENTS.md](../AGENTS.md).

### Kinds

- **repair**: the editor starts empty. The default.
- **fault**: `seed` holds the region with a planted runtime or logic fault. It must compile, then panic, deadlock or print the wrong output.
- **refit**: `seed` holds a working region that breaks a construct rule in `check.requires` or `check.forbids`. It must already print the target.
- **decode**: `template.txt` is the whole program, with no markers. The player types its output. Set `check.prediction: true` and put the output in `check.stdout_equals`.

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
  - **compute-and-print**: markers inside `main`; the player computes *and* prints.
  - **compute-only**: the footer ends with `fmt.Println(result)` referencing a value
    the player must define. This pins the output format and forces the lesson (as
    above, `fmt.Println(celsius)` forces declaring `celsius`). Prefer it when there is
    one clear result.
- Package-level work (functions, types, methods, interfaces) goes with the markers at
  package level and a `main()` in the footer that exercises it.
- Input data the answer computes from may sit between `// === INPUT ===` and
  `// === END INPUT ===` in the header. `check.cases` swaps other data into that block
  (see §4).
- Blocked imports (`os/exec`, `syscall`, `unsafe`, `plugin`, `golang.org/x/sys`) are
  rejected for safety: generated code runs automatically during checking.

## 4. Checks

`check` decides pass/fail against the program's stdout (exit must be clean):

| field | meaning |
|-------|---------|
| `stdout_equals`   | trimmed stdout must equal this (preferred: deterministic, teaches exact output) |
| `stdout_contains` | stdout must contain this substring |
| `stdout_nonempty` | any non-empty stdout passes (reserve for open beats, e.g. "declare your name") |
| `cases`           | list of `{input, stdout}`: the input block is replaced and the program must print that stdout |
| `requires`        | construct rules the player's region must use, e.g. `range`, `call:strings.Fields` |
| `forbids`         | construct rules the region must not use, e.g. `for-classic` |
| `prediction`      | decode mission: the player's typed text is compared with `stdout_equals` |

Prefer `stdout_equals`. Concurrency missions (goroutines, channels) must produce
**deterministic** output: sort or count before printing, or the check can't hold.
Rule forms are listed in [MISSION_PROTOCOL.md](MISSION_PROTOCOL.md#construct-rules).

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

   `levels/loader_test.go` runs every bundled `answer` through the judge against its
   own template and check, and confirms that a `seed` fails. A mission whose answer
   does not pass breaks the test.

To start from a forged draft instead of a blank page:

```sh
goscii forge my-track --topic maps --kind fault --difficulty easy --count 4
```

It writes `levels/adventures/my-track/NN_<beat>/` with a verified `level.yaml` and
`template.txt` per mission. Edit the stories, then register and test as above.

## 6. Authoring conventions

- One new idiom per mission; carry a small but genuine algorithm, not just syntax.
- Two hints: one conceptual, one concrete-syntax, never the full answer.
- Keep the narrator's voice in the `story`; keep the task unambiguous.
- Map each mission to an in-world beat so the track reads as one story, not a worksheet.
