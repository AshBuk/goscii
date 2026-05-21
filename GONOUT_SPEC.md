# GOSCII — Specification v0.1

## Concept

A terminal game for learning Go programming language with narrative in ASCII world.

The player helps an astronaut navigate through space by writing real Go code. Each level teaches a new language concept — from variables to concurrency. Fully interactive in the terminal: no external editor required.

### AI Companion — AXIS

The astronaut is not alone. AXIS (Adaptive eXecution Intelligence System) is the onboard AI — a half-broken companion who survived the crash alongside the player. AXIS speaks through the terminal: delivers missions, reacts to the player's code, hints when asked, and celebrates breakthroughs.

AXIS starts fragmented — short sentences, corrupted memory, static. As the player progresses, AXIS recovers: more coherent, more personality, occasional dry humor. The tone shifts from eerie survival to a working partnership.

AXIS is the narrative voice of every `story:` field in level.yaml.

---

## Game Modes

### 1. Onboarding (main campaign)

Linear story campaign covering all Go fundamentals:

```
Variables -> Functions -> Conditionals -> Loops ->
Structs -> Interfaces -> Errors -> Goroutines -> Channels
```

Each concept: 2-4 levels. Total target: ~30 levels.

### 2. Adventures (additional missions)

Standalone mini-games with their own narrative and mechanics. Each adventure focuses on a specific topic or challenge.

Examples:
- "Cargo Run" — slices and maps puzzle
- "Signal Tower" — goroutines and channels
- "Debug Crater" — find and fix bugs in broken code

Adventures can be added as external YAML packs in the future.

---

## Name & CLI

- Binary: `goscii`
- Commands:
  - `goscii start` — start or continue onboarding campaign
  - `goscii start --adventure cargo-run` — start a specific adventure
  - `goscii progress` — show completed levels
  - `goscii reset` — reset current level
  - `goscii reset --all` — reset all progress

---

## Core Loop

```
Show level screen (ASCII world + mission text)
    |
Player writes Go code in terminal textarea
    |
Player presses Button or Hotkey to run
    |
Code is injected into template -> go run
    |
Parse stdout/stderr
    |
Pass -> victory animation -> next level
Fail -> error message + optional hint (press 'h')
```

---

## Screen Layout

```
+------------------------------------------------------+
|  GOSCII          Level 3/20         [===...]     |
+------------------------------------------------------+
|                                                      |
|   (moon)  . . . . [astronaut] . . . . . . . [base]   |
|                                                      |
|   Mission: the astronaut needs fuel.                 |
|   Declare a variable `fuel` with value 42.           |
|                                                      |
+------------------------------------------------------+
|  > var fuel = 42                                     |
|  > _                                                 |
|                                                      |
+------------------------------------------------------+
|  [Ctrl+Enter] run   [h] hint   [n] skip   [q] quit  |
+------------------------------------------------------+
```

---

## Project Structure

```
goscii/
├── main.go
├── cmd/                        <- Cobra commands
│   ├── root.go
│   ├── start.go
│   ├── progress.go
│   └── reset.go
├── engine/
│   ├── runner.go               <- os/exec, go run, capture output
│   ├── checker.go              <- compare output vs expected
│   └── state.go                <- load/save progress JSON
├── tui/
│   ├── game.go                 <- main BubbleTea model
│   ├── editor.go               <- textarea input component
│   └── world.go                <- ASCII world renderer
├── levels/
│   ├── onboarding/
│   │   ├── 01_variables/
│   │   │   ├── level.yaml
│   │   │   └── template.go
│   │   ├── 02_functions/
│   │   ├── 03_conditionals/
│   │   ├── 04_loops/
│   │   ├── 05_structs/
│   │   ├── 06_interfaces/
│   │   ├── 07_errors/
│   │   ├── 08_goroutines/
│   │   └── 09_channels/
│   └── adventures/
│       ├── cargo-run/
│       └── signal-tower/
└── assets/
    └── astronaut/              <- ASCII sprite frames
        ├── idle.txt
        ├── walking.txt
        ├── celebrate.txt
        └── crash.txt
```

---

## Player State

A small JSON state file at `~/.goscii/progress.json` persists between levels:

```json
{
  "current_level": "02_variables_2",
  "vars": {
    "player_name": "nova"
  }
}
```

Level narratives can reference player vars via `{{ .Vars.player_name }}`:

```yaml
story: |
  "Welcome back, {{ .Vars.player_name }}. The reactor needs attention."
```

The `capture` field in `level.yaml` writes stdout into a named var on success.

---

## Level Format

### level.yaml

```yaml
id: 01_variables_1
title: "Who Am I"
concept: variables
story: |
  The ship went down hard.
  You crawl out of the wreckage into the dust.
  Systems offline. Memory corrupted.
  The onboard AI speaks in a broken voice:
  "Identity unknown. Please... declare your name."
hint: "In Go: `var name = value` or short form `name := value`"
check:
  stdout_nonempty: true
capture:
  player_name: stdout
```

### template.go

```go
package main

import "fmt"

func main() {
    // === YOUR CODE HERE ===

    // === END ===
    fmt.Println(name)
}
```

---

## Tech Stack

| Layer          | Library                                        |
|----------------|------------------------------------------------|
| TUI framework  | github.com/charmbracelet/bubbletea             |
| Styling        | github.com/charmbracelet/lipgloss              |
| Text input     | github.com/charmbracelet/bubbles/textarea      |
| CLI commands   | github.com/spf13/cobra                         |
| Code execution | os/exec -> go run                              |
| Progress       | JSON file at ~/.goscii/progress.json       |

---

## Astronaut States

| State      | Trigger                        |
|------------|-------------------------------|
| idle       | waiting for input              |
| walking    | code submitted, running        |
| celebrate  | level passed                   |
| crash      | compile error or wrong output  |

---

## Out of Scope (v0.1)

- Multiplayer / leaderboard
- In-terminal syntax highlighting (nice to have in v0.2)
- Languages other than Go

---

## Open Questions

- Strict linear progression or allow skipping levels?
- Show raw Go compiler errors or wrap them in friendly messages?
- Syntax highlighting in textarea via `glamour`?
