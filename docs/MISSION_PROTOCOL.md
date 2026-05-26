# MISSION PROTOCOL

*GOSCII core log. Mission protocol documentation recovered*

---

## Transmission Contract

GOSCII generates missions. Each mission is a repair job on a failing station system.
The astronaut writes Go. GOSCII verifies the output.


## Signal Clearance Levels

Four levels. Each determines how much GOSCII reveals.

| Level | Transmission target shown | Story tone | Hints |
|---|---|---|---|
| easy | yes | plain English, unambiguous | available |
| medium | yes | scenario-driven, goal clear | available |
| hard | yes | full GOSCII voice, fragmented | available |
| survival | **no** | corrupted, ambiguous by design | **disabled** |

`Difficulty` is a mission concept — lives in `levels`.

---

## Transmission Target

After the story, before the editor: `SIGNAL ▸ target output → <value>`

This is the contract. The exact transmission GOSCII expects back.

On survival: silence. The astronaut figures it out alone.

**Output ownership rule:** the template never prints anything after `// === END ===`.
All output comes from the astronaut's code. Story never asks to print what the scaffold already prints.
One output owner. Always.

---

## Story Protocol by Level

- **easy** — plain instruction. State what to declare, what to print. No riddles. Light sci-fi flavor permitted. End with `Output: <value>`
- **medium** — task through scenario. Goal still clear. End with `Output: <value>`
- **hard** — full GOSCII voice: fragmented, terse, survival tone. Difficulty from the Go concept. End with `Output: <value>`
- **survival** — maximally corrupted. No `Output:` line. Ambiguous by design.

---

## Signal Validation

GOSCII validates its own transmissions before sending.

After `Generate()`, before the astronaut sees the mission:
1. Run the reference answer through the engine
2. Verify it satisfies the check
3. If it fails — regenerate. Up to 3 attempts.
4. If all fail: `GOSCII signal corrupted after 3 attempts`

A broken mission is worse than a hard one. The astronaut must always be able to pass.

---

## Mission Chains

Multiple repairs in sequence.

Each mission in a chain builds on the last — same station system, escalating Go concept.
GOSCII receives the previous story and the astronaut's actual code before generating the next mission.
Only the last 3 entries are passed to avoid signal overload.

| Level | Chain length |
|---|---|
| easy | 5 |
| medium | 10 |
| hard | 20 |
| survival | 20 |

Chain progress shown in the cockpit header: `[3/10]`

`MissionChain` tracks the sequence. `chain.Done()` ends it. `chain.Brief()` builds the next transmission brief.

---

## Survival Protocol

Survival is not hard with hints removed. It is a separate mode — intentional, earned.

The astronaut receives a corrupted brief. No transmission target. No hints. Scaffold minimal.
The answer key exists but is not shown until requested.
GOSCII checks the output internally. The astronaut never sees what GOSCII expects.

This is what the station feels like at the end.
