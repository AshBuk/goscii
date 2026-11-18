# goscii

A terminal game for learning Go programming language with sci-fi narrative in ASCII world.

The player helps an astronaut navigate through space by writing real Go code. Each level teaches a new language concept, from variables to concurrency. Fully interactive in the terminal: no external editor required.

---

## Naming conventions

Use the sci-fi vocabulary, not generic game engine

### Variables track the type

Short names keyed to the type initial.

```go
m *levels.Mission   // not l, not mission
c Cockpit           // receiver on Cockpit methods
a *ai.Anthropic     // receiver on Anthropic methods
a levels.Adventure  // receiver on Adventure methods
```

### Methods read as sentences

Design method calls so the full expression reads as a phrase.

```go
levels.Adventure("onboarding").Missions()  // ✓ "missions of this adventure"
forge.Forge(ctx, signal, order, report)    // ✓ "forge a mission from the order"
chain.Next(progress)                       // ✓ "next order of this chain"
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

- `levels.Mission`: the playable unit (story, hints, check, answer, kind, seed)
- `tui.Cockpit`: the main mission screen (editor, run, hints, comms, status)
- `ai.Topic`: a Go concept category shown in the mission popup; `ai.Beat` is one idiom inside it
- `ai.Provider`: the transport interface; "signal" is its name in cmd-layer code
- `ai.Band`: a provider's verified and retired models, in `ai/catalog.go`
- `forge`: the mission harness (order, assay, repair loop, chain, archive, bench)
- `engine`: stays generic/technical; no sci-fi naming required here

The harness owns state, randomness and truth; the model owns text and code. Expected output always comes from running the reference answer. See docs/MISSION_PROTOCOL.md.
