# GOSCII Architecture

GOSCII is a terminal Go-learning game built around one execution contract: a mission provides a Go template, the player fills the marked section, and the engine runs the resulting file with `go run`.

The product concept and narrative live in `docs/GOSCII_CONCEPT.md`. This file describes the technical shape of the application.

---

## System Shape

```
CLI command
  -> setup / selection TUI
  -> mission source
       -> AI provider
       -> embedded adventure level
  -> game TUI
  -> engine.RunCode
  -> progress/config storage
```

The TUI does not know where a mission came from.
Offline adventures and AI-generated practice both end as the same pair:

```
*levels.Level + Go template string
```

---

## Runtime Flows

### AI Practice

```
goscii start
  -> load config
  -> first-run setup if API key is missing
  -> choose topic
  -> choose difficulty
  -> Provider.Generate(ctx, Request)
  -> run game
  -> pass: record topic completion
  -> ctrl+n: generate another mission for the same topic/difficulty
```

### Offline Adventure

```
goscii start --adventure onboarding
  -> load embedded YAML level + template
  -> run game without AI provider
```

---

## Packages

```
cmd/       Cobra commands and top-level orchestration.
ai/        Provider interface, topic catalog, prompts, Groq client.
tui/       Bubble Tea models: setup, entry selector, game screen, world render.
engine/    Code injection, execution, output checks, config/progress storage.
levels/    Embedded handcrafted levels.
assets/    Embedded ASCII sprites.
```

`cmd/start.go` is the main coordinator. It wires config, selection, mission
generation/loading, game execution, and progress persistence.

---

## Mission Contract

Every mission has:

- `levels.Level`: metadata, story, hints, answer, and check rule.
- `template`: a full Go file with player-code markers.

Required template markers:

```go
// === YOUR CODE HERE ===
// === END ===
```

The engine injects player code between those markers, writes a temporary `.go`
file, runs `go run`, then verifies stdout/stderr with `engine.CheckRule`.

Supported checks:

- `stdout_nonempty`
- `stdout_contains`
- `stdout_equals`

---

## AI Boundary

AI is isolated behind `ai.Provider`:

```go
type Provider interface {
	Generate(ctx context.Context, req Request) (*levels.Level, string, error)
	Analyze(ctx context.Context, req AnalyzeRequest) (string, error)
	Name() string
	Models() []string
}
```

Current implementation:

- Groq only.
- OpenAI-compatible HTTP API via `net/http`.
- Generated JSON is parsed, normalized, and validated before it reaches the TUI.

The TUI receives only `*levels.Level`, `template`, and an optional provider for
log analysis.

---

## Execution Boundary

`engine.RunCode` is intentionally small:

1. Inject player code into the template.
2. Write a temporary Go file.
3. Run `go run` with a timeout.
4. Capture stdout/stderr.
5. Return `RunResult`.

Player code is not sandboxed beyond process isolation and timeout. Treat this as a local learning tool, not a hostile-code execution environment.

---

## Persistent Data

Data lives under the platform data directory:

```
$XDG_DATA_HOME/goscii/
~/.local/share/goscii/
```

### `config.json`

```json
{
  "provider": "groq",
  "model": "llama-3.3-70b-versatile",
  "api_key": "gsk_..."
}
```

Saved with `0600` permissions.

### `progress.json`

```json
{
  "adventure_checkpoint": "onboarding/02_functions",
  "vars": {
    "player_name": "nova"
  },
  "topics": {
    "variables": {
      "completed": [
        {
          "level_id": "variables-easy-a3f7k2",
          "difficulty": "easy",
          "completed_at": "2026-05-25T00:00:00Z"
        }
      ]
    }
  }
}
```

`goscii progress` intentionally shows only solved mission counts by topic.

---

## TUI States

```
idle       waiting for player input
running    code submitted
passed     check succeeded
failed     compile/runtime/check failed
analyzing  AI is explaining the last failure
```

Controls stay local to the game screen:

```
ctrl+r  run
ctrl+h  hint
ctrl+g  analyze failure
ctrl+a  show answer
ctrl+n  next mission after pass
ctrl+c  quit
```

---

## Design Rules

- Keep AI generation outside the TUI.
- Keep mission validation before gameplay.
- Keep code execution inside `engine`.
- Keep progress simple: per-topic completion counts first.
- Prefer small provider-specific code over premature multi-provider abstraction.
- Add new adventure content through `levels/`, not command branching.

---

## Extension Points

- Add providers by implementing `ai.Provider` and wiring setup/runtime selection.
- Add topics in `ai/topics.go`.
- Add handcrafted adventures under `levels/`.
- Add richer checks by extending `engine.CheckRule`.
- Add replay/caching only after generated missions prove useful enough to keep.

