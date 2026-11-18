# GOSCII - Concept

A terminal game for learning Go programming language with sci-fi narrative.

The player helps an astronaut navigate through space by writing real Go code. Two modes: a linear story campaign (offline, bundled) and AI-generated practice missions for any Go topic on demand.

### AI Companion - GOSCII

The astronaut is not alone. GOSCII (Go Orbital Survival Coding Interactive Interface) is the onboard AI: a half-broken companion who survived the crash alongside the player. GOSCII speaks through the terminal: delivers missions, reacts to the player's code, hints when asked, and celebrates breakthroughs.

GOSCII starts fragmented: short sentences, corrupted memory, static. As the player progresses, GOSCII recovers: more coherent, more personality, occasional dry humor. The tone shifts from eerie survival to a working partnership. The shift follows the number of repaired missions (see [MISSION_PROTOCOL.md](MISSION_PROTOCOL.md#station-and-crew-logs)).

GOSCII is the narrative voice of every `story:` field in level.yaml and every AI-generated mission. On the comms line (`ctrl+g` in the cockpit) the player can ask it about their code at any time.

### The Station

The station has 14 systems, from atmosphere to propulsion. Each starts damaged. Every solved mission repairs the system it was set on, and the hub's Station screen shows the integrity bars. Finishing a mission chain decrypts one crew log: fragments of what happened to the Halcyon-7 crew before the crash. The facts live in `levels/logs.yaml`, so the story stays the same for every player.

---

## Game Modes

### 1. AI Practice (primary mode)

The player selects a Go topic, a mission kind and a difficulty level. An AI provider forges a unique coding mission every session, different each time, with hints and a reference answer that GOSCII test-fires before the player sees it.

```
Hub: Difficulty (easy / medium / hard / survival), set once
Hub: Mission
  -> Mission popup (topic, or Open channel for a free-text wish)
  -> Kind popup (mixed / repair / fault / decode / refit / incident)
  -> GOSCII forges and test-fires the mission
  -> Cockpit
  -> Pass: topic counter +1, station system repaired
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
| generics | Generics |
| iterators | Iterators |
| strings | strings & strconv |
| time | time |
| slog | log/slog |

Each topic splits into beats, one idiom each (`comma-ok`, `wg-go`, `early-stop`, …). GOSCII picks the beat the player has practised least.

Every forged mission is saved to the archive and can be replayed offline from the hub.

### 2. Onboarding and Other Handmade Campaigns

A linear story campaign covering Go fundamentals, bundled as YAML levels:

Variables -> Functions -> Conditionals -> Loops ->
Structs -> Interfaces -> Errors -> Goroutines -> Channels

29 levels. Each concept opens with a repair mission and follows with fault, decode or refit missions on the same idiom. A renumbered track keeps the player's checkpoint: it matches by the name after the `NN_` prefix.

Entry point: `goscii start` → hub menu → Adventure

Custom adventures can follow the same structure or focus on a specific topic. `goscii forge` drafts one with the wired signal.

---

## Name & CLI

- Binary: `goscii`
- Commands:
  - `goscii start`: open hub: configure signal, pick difficulty, launch mission or adventure
  - `goscii adventure [name]`: run a bundled adventure directly
  - `goscii progress`: show solved missions by topic and station integrity
  - `goscii logs`: read the crew logs recovered so far
  - `goscii reset`: reset adventure checkpoint
  - `goscii reset --all`: reset all progress
  - `goscii signal models`: list the models the wired key can reach
  - `goscii signal bench`: forge a batch of missions and report first-try pass rate, repairs, time and tokens
  - `goscii forge <track>`: forge a draft adventure track
