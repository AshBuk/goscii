# Visual Protocol

## Design Principles

1. **Centered layouts** — all screens align content to the horizontal center of the terminal using `lipgloss.Width(w).Align(lipgloss.Center)`. Width comes from `tea.WindowSizeMsg`, already tracked in every model.
2. **Single theme source** — `tui/theme.go` owns every color, style, and the ASCII logo. Nothing else defines colors.
3. **No new deps** — hardcode the GOSCII logo as a `const` string. One fewer dependency is always better.
4. **Consistent spacing** — 1 blank line after the logo, 1 blank line before key hints, no double-blank lines mid-menu.
5. **Subtlety over spectacle** — animations are opt-in and slow (≥ 600ms tick). They must not create visual noise during keyboard interaction.

---

## Theme — `tui/theme.go`

### Color palette

Light-purple (lavender) palette:

| Token | Color (256) | Usage |
|---|---|---|
| `colorAccent` | `141` (light purple) | logo, selected item, progress fill, menu border |
| `colorPulse` | `183` (bright lavender) | logo pulse |
| `colorText` | `189` (pale lavender) | active menu items, labels |
| `colorMuted` | `146` (muted lavender) | inactive items, separators |
| `colorDim` | `103` (slate purple) | badges, secondary values |
| `colorHint` | `97` (purple-grey) | key hint bar |
| `colorWarn` | `215` (soft amber) | "not wired" warning |
| `colorPass` | `120` (soft mint green) | mission passed state |
| `colorFail` | `210` (soft coral) | mission failed, crash state |

### ASCII logo — `gosciiLogo`

Box-drawing style, 6 lines tall:

```
 ██████╗  ██████╗ ███████╗ ██████╗██╗██╗
██╔════╝ ██╔═══██╗██╔════╝██╔════╝██║██║
██║  ███╗██║   ██║███████╗██║     ██║██║
██║   ██║██║   ██║╚════██║██║     ██║██║
╚██████╔╝╚██████╔╝███████║╚██████╗██║██║
 ╚═════╝  ╚═════╝ ╚══════╝ ╚═════╝╚═╝╚═╝
```

Rendered with `colorAccent`, centered via `lipgloss.Center`.

---

## Screens

### 1. Hub — `tui/hub.go`

**Layout (top → bottom):**
```
[blank line]
[GOSCII logo — centered, colorAccent]
[subtitle: "Go Orbital Survival Coding Interactive Interface" — centered, colorDim]
[blank line]
[menu items — centered block]
  > Signal       groq · mixtral-8x7b-32768
    Mission      select topic →
    Difficulty   Easy
    Adventure    onboarding
    ──────────────────────────
    Quit
[blank line]
[key hints — centered, colorHint]
```

**Menu centering:** items are rendered into a fixed-width block (e.g. 36 chars), then that block is centered in the terminal via `lipgloss.Place` or manual padding.
