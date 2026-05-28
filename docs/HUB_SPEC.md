# Hub Spec

Navigation hub for `goscii start` — a single persistent TUI screen that routes the user to all modes.

---

## Naming

| Symbol | In-world label |
|---|---|
| `tui.Hub` | the main menu screen |
| `tui.HubModel` | the Bubble Tea model |
| `tui/hub.go` | file |
| `BackMsg` | message emitted by child screens to return to hub |

---

## Navigation architecture

Single `tea.Program` for the full lifecycle. `cmd/start.go` launches one `tui.HubModel` — a router that holds the active child screen. No outer loop.

```
goscii start
└─ HubModel  ←────────────────────────────────┐
   ├─ Signal      → ConfigModel  ──────────────┤ BackMsg
   ├─ Mission     → EntryModel (topic only)     │
   │               → GenerateModel (progress)   │
   │               → Cockpit ──────────────────┤
   ├─ Difficulty  → inline sub-step in Hub      │
   ├─ Adventure   → adventure runner ───────────┤
   └─ Quit        → tea.Quit                    │
                                                │
   Ctrl+C in any child screen ─────────────────┘
   Ctrl+C in HubModel → tea.Quit
```

---

## HubModel — menu layout

**Signal configured:**
```
  GOSCII — MISSION CONTROL

  > Signal       Groq · llama-3.3-70b
    Mission      Variables & Types
    Difficulty   Easy
    Adventure    onboarding
    ──────────────────────────
    Quit

  [↑/↓] navigate   [enter] select   [ctrl+c] quit
```

**Signal not configured:**
```
  > Signal       not wired ⚠
    Mission      · wire signal for mission protocols ·
```

Mission item is dimmed; cursor skips it automatically. Pressing `Enter` on it shows an inline hint and jumps cursor to Signal.

---

## Difficulty — inline sub-step

`Enter` on Difficulty transitions `hubStep` inline — no child model launched:

```
  Select difficulty:

  > Easy
    Medium
    Hard

  [enter] confirm   [esc] back
```

Selection persists in HubModel state, displayed in the main menu row.

---

## GenerateModel — new screen

Replaces the current `fmt.Printf` stdout output during AI mission generation. Shown between EntryModel and Cockpit.

**In progress:**
```
  Wiring Easy signal for "Variables & Types"  [1/2]

  Validating  [████████░░░░░░░░]  3 / 8

  [ctrl+c] abort
```

Progress bar fills one segment per validation attempt (0 → 8). On success — instant transition to Cockpit.

**All 8 attempts failed:**
```
  Brain module failure.
  Mission protocol could not be formed.

  [R] retry   [M] new mission   [H] hub
```

---

## Ctrl+C contract

| Screen | Ctrl+C |
|---|---|
| Cockpit | → HubModel |
| ConfigModel | → HubModel |
| EntryModel | → HubModel |
| GenerateModel | → HubModel |
| HubModel | → `tea.Quit` |

Child screens emit `BackMsg{}` instead of `tea.Quit`. HubModel intercepts it, sets active child to `nil`, resumes hub menu.

---

## File map

| File | Change |
|---|---|
| `tui/homehub.go` | **new** — HubModel router, menu, inline Difficulty sub-step |
| `tui/generate.go` | **new** — GenerateModel with progress bar, failure screen |
| `tui/entry.go` | remove `stepDifficulty` — topic selection only |
| `tui/config.go` | Esc on first step → `BackMsg` instead of no-op |
| `tui/cockpit.go` | Ctrl+C → `BackMsg` instead of `tea.Quit` |
| `cmd/start.go` | replace all of `runStart` with `tea.NewProgram(tui.NewHub(...))` |
