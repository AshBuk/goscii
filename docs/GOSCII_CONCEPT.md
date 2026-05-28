# GOSCII - Concept

A terminal game for learning Go programming language with narrative in an ASCII world.

The player helps an astronaut navigate through space by writing real Go code. Two modes: a linear story campaign (offline, bundled) and AI-generated practice missions for any Go topic on demand.

### AI Companion - GOSCII

The astronaut is not alone. GOSCII (Go Orbital Survival Coding Interactive Interface) is the onboard AI - a half-broken companion who survived the crash alongside the player. GOSCII speaks through the terminal: delivers missions, reacts to the player's code, hints when asked, and celebrates breakthroughs.

GOSCII starts fragmented - short sentences, corrupted memory, static. As the player progresses, GOSCII recovers: more coherent, more personality, occasional dry humor. The tone shifts from eerie survival to a working partnership.

GOSCII is the narrative voice of every `story:` field in level.yaml and every AI-generated mission.

---

## Game Modes

### 1. AI Practice (primary mode)

The player selects a Go topic and difficulty level. An AI provider generates a unique coding mission every session — different each time, with hints and a reference answer.

```
Topic selector
  -> Difficulty (easy / medium / hard)
  -> AI generates mission (Mission + template + hints + answer)
  -> Cockpit
  -> Pass: topic counter +1 with difficulty logged
```

Supported topics (stdlib focus):

| Slug | Title |
|------|-------|
| variables | Variables & Types |
| functions | Functions |
| control-flow | Control Flow |
| slices | Slices |
| maps | Maps |
| structs | Structs & Methods |
| interfaces | Interfaces |
| errors | Errors |
| goroutines | Goroutines |
| channels | Channels |
| select | Select |
| context | Context |
| io | io & Readers |
| http | net/http |
| json | encoding/json |
| files | os & Files |

### 2. Onboarding and Other Handmade Campaigns

A linear story campaign covering Go fundamentals, bundled as YAML levels:

Variables -> Functions -> Conditionals -> Loops ->
Structs -> Interfaces -> Errors -> Goroutines -> Channels

Each concept has 2-4 levels. Target total: ~30 levels.

Entry point: `goscii start` → hub menu → Adventure

Custom adventures can follow the same structure or focus on a specific topic.
---

## Name & CLI

- Binary: `goscii`
- Commands:
  - `goscii start` — open hub: configure signal, pick difficulty, launch mission or adventure
  - `goscii progress` — show solved missions by topic
  - `goscii reset` — reset adventure checkpoint
  - `goscii reset --all` — reset all progress

