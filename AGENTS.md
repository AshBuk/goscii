# goscii

A terminal game for learning Go programming language with narrative in ASCII world.

The player helps an astronaut navigate through space by writing real Go code. Each level teaches a new language concept - from variables to concurrency. Fully interactive in the terminal: no external editor required.

---

## Naming conventions

Use the sci-fi vocabulary, not generic game engine

### Variables track the type

Short names keyed to the type initial.

```go
m *levels.Mission   // not l, not mission
c Cockpit           // receiver on Cockpit methods
g *ai.Groq          // receiver on Groq methods
a levels.Adventure  // receiver on Adventure methods
```

### Methods read as sentences

Design method calls so the full expression reads as a phrase.

```go
levels.Adventure("onboarding").Missions()  // ✓ "missions of this adventure"
signal.Generate(ctx, req)                  // ✓ "signal generates a mission"
```

### User-facing strings use GOSCII voice

Error messages, UI labels, and placeholders stay in-world.

```go
"GOSCII signal lost."             // not "provider error"
"mission timed out"               // not "execution timeout"
"signal %q is not wired yet"      // not "provider not supported"
"Select signal:"                  // not "Select provider:"
```

### What belongs where

- `levels.Mission` — the playable unit (story, hints, check, answer)
- `tui.Cockpit` — the main mission screen (editor, run, hints, status)
- `ai.Topic` — a Go concept category shown in the topic selector
- `ai.Provider` — the interface; "signal" is its name in cmd-layer code
- `engine` — stays generic/technical; no sci-fi naming required here
