# MISSION PROTOCOL

*GOSCII core log. Mission protocol documentation recovered*

---

## Transmission Contract

GOSCII forges missions. Each mission is a repair job on a failing station system.
The astronaut writes code. GOSCII verifies the output.

The split of duties is fixed. The harness owns state, randomness and truth: which idiom to teach, which system is damaged, what the correct output is. The model owns text and code: the story, the hints, the program. The model never states the expected output. GOSCII runs the reference answer and records what it prints.

Code lives in `forge/` (order, assay, forge loop, archive, bench), `engine/` (build, run, judge, construct rules) and `ai/` (signal transport, topics, partner prompts).

---

## Signal Clearance Levels

Four levels. Each determines how much GOSCII reveals.

| Level | Transmission target shown | Story tone | Hints |
|---|---|---|---|
| easy | yes | plain English, unambiguous | available |
| medium | yes | scenario-driven, goal clear | available |
| hard | yes | full GOSCII voice, fragmented | available |
| survival | **no** | corrupted, ambiguous by design | **disabled** |

`Difficulty` is a mission concept and lives in `levels`.

---

## Mission Kinds

| Kind | Editor starts with | The astronaut | Check |
|---|---|---|---|
| repair | empty region | writes the missing code | output, cases, construct rules |
| fault | the faulty region | finds and fixes the planted fault | output, cases, construct rules |
| decode | nothing editable | types the predicted output | prediction equals the real run |
| refit | a clumsy working region | rewrites it to follow the idiom | output, cases, construct rules |

`levels.Kind` names them. A chain without a fixed kind rotates repair, decode, fault, repair, refit, fault.

---

## Orders

The harness writes an order before the model writes a word. `forge.Order` carries the topic, one beat (a single idiom such as `comma-ok` or `wg-go`), the kind, the difficulty, the station system, the pilot dossier and the chain brief.

The beat is the least practised one in the pilot's record, skipping beats already used in the chain. Hard and survival lean toward later beats. Refit only takes beats with a construct rule. The system is one of the three most damaged on the station that the chain has not touched yet.

The dossier tells the model what the pilot has done on this topic and which beats took many runs, so those parts stay gentle.

---

## Transmission Target

After the story, before the editor: `SIGNAL ▸ target output → <value>`

This is the contract. The exact transmission GOSCII expects back. The value comes from running the reference answer, never from the model's text. Any `Output:` line the model writes into the story is stripped.

On survival: silence. The astronaut figures it out alone.

**Output ownership rule:** the template never prints anything after `// === END ===` that the astronaut must produce.
With the region empty, the scaffold must not print the target output by itself; such a draft is rejected.

**Marker placement:** the two markers sit wherever the astronaut writes.
Compute/print tasks place them inside `func main()`; tasks that define types,
methods or functions place them at package (top) level, and the astronaut writes
the declarations and `func main()` between them. Code outside the markers must
be balanced on its own (no block left open above `// === YOUR CODE HERE ===`).
The cockpit renders everything above the first marker as the read-only header
and everything below `// === END ===` as the read-only footer. No braces are
synthesized.

**Input block:** a template may hold telemetry between `// === INPUT ===` and `// === END INPUT ===`. The model sends alternative inputs for that block. GOSCII runs the reference on each one and keeps up to 3 that build, run and change the output. They become `check.cases`. A player who hardcodes the printed answer passes the main reading and fails a shifted one. A variable with a literal value that the region reads must sit in the input block, and at least one alternative must change the output; otherwise the draft goes back for repair. The player never sees the alternative inputs, so every number and word in the reference region's literals must appear in the story, the scaffold, the seed or the sample output (not in survival). Numbers 0 to 2 and arguments passed straight to a library call or `make` are exempt. A failed case shows the player its input and what their code printed.

---

## Construct Rules

A beat may require or forbid constructs in the astronaut's region. GOSCII checks them on the AST, not with text search. Rule forms:

| Form | Example | Matches |
|---|---|---|
| keyword | `for`, `range`, `go`, `comma-ok`, `generics` | a construct kind (full list in `engine/rules.go`) |
| package member | `sync.WaitGroup`, `errors.*` | a selector on an imported package |
| method or field | `.Lock` | a selector on any value |
| call | `call:append`, `call:slices.Sort`, `call:.Wait` | a call to a builtin, package function or method |

The cockpit lists the rules under PROTOCOL. A run that prints the right output but breaks a rule fails with an off-protocol message. The reference answer must follow the rules too.

---

## Signal Validation

GOSCII validates its own transmissions before sending. The model replies with one JSON object that follows a strict schema per kind. `forge.assay` then checks the draft in this order:

1. The reply parses, has a title, story, hints and solution, imports nothing blocked, and the solution parses as Go.
2. The region markers appear once each, in order, with a non-empty region between them.
3. The scaffold (the solution with the region removed) parses as Go on its own. This rejects templates that split a syntactic unit across the markers (open `import (`, `struct {`, a call): they compile only because the reference answer closes them, but leave the astronaut an incoherent editing surface.
4. The reference builds once and runs three times. It must exit cleanly, print something, and print the same thing on every run.
5. The reference follows the beat's construct rules.
6. The empty scaffold does not print the target by itself.
7. Fault: the faulty region compiles and then panics, deadlocks or prints wrong output. Refit: the seed already prints the target and breaks a construct rule. Decode: the program has no markers, prints at most 8 lines, and uses the beat's required constructs.

A failed check returns a defect with the exact compiler error, stack trace or diff. GOSCII sends it back to the model as a repair turn in the same conversation, up to 2 repair rounds per draft and 3 drafts per mission. A truncated or refused reply starts a new draft. Any other transport failure ends the forge.

If all drafts fail: `GOSCII could not forge a valid mission after 3 drafts`, with the last defect.

A broken mission is worse than a hard one. The astronaut must always be able to pass.

---

## Mission Chains

Multiple repairs in sequence.

Each mission in a chain builds on the last: escalating idiom, a different damaged system, the same narrative thread.
GOSCII receives the previous stories and the astronaut's actual code before forging the next mission.
Only the last 3 entries are passed to avoid signal overload.

| Level | Chain length |
|---|---|
| easy | 3 |
| medium | 6 |
| hard | 12 |
| survival | 12 |

The last mission of a chain is a capstone: it combines the earlier idioms, and hints are off.

**Incident mode** keeps one system for the whole chain. Every step is a repair, and the astronaut's previous program becomes the next scaffold.

**Prefetch:** after the first run of a mission, GOSCII forges the next one in the background, planned as if the current reference answer were the astronaut's code. Incident mode skips it, since the next scaffold depends on the real code. Set `"prefetch": false` in `config.json` to turn it off.

`forge.Chain` tracks the sequence. `Next` plans the current step and returns the same order on a retry. `Done` ends the chain.

---

## Station and Crew Logs

Every station system starts at 20% integrity. A solved mission repairs its system by 6 on easy, 10 on medium, 14 on hard and 20 on survival, capped at 100. Finishing a chain lets GOSCII decrypt one crew log. The facts in `levels/logs.yaml` are canon; the model only writes them up in GOSCII's voice.

GOSCII's voice follows the number of repaired missions: fragmented at first, steadier after 10, a working partner after 30.

---

## Survival Protocol

Survival is not hard with hints removed. It is a separate mode: intentional, earned.

The astronaut receives a corrupted brief. No transmission target. No hints. Scaffold minimal.
The answer key exists but is not shown until requested.
GOSCII checks the output internally. The astronaut never sees what GOSCII expects.

Life support is finite: 4 runs of oxygen per mission, shared across the chain. When it runs out, the chain ends.

This is what the station feels like at the end.
