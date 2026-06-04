// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

/*
GOSCII is a sci-fi TUI for learning Go — the Go Orbital Survival Coding
Interactive Interface.

You're a on a crashed orbital station, and the only way home
is through the code: every broken system is a Go exercise. The onboard AI, also
called GOSCII, hands you missions, checks your output, and pieces its own
corrupted memory back together as you progress.

Two ways to play:

  - Offline adventures — bundled mission tracks, no API key. You write Go, GOSCII
    compiles and runs it, and checks the output.
  - AI missions — bring your own key for an endless stream of generated exercises
    by topic and difficulty, wrapped in the same narrative.

Commands:

	goscii start              # launch the home hub
	goscii adventure [name]   # play a bundled offline track directly
	goscii progress           # show solved missions by topic
	goscii reset [--all]      # reset the adventure checkpoint, or wipe all progress

Full briefing: https://github.com/AshBuk/goscii
*/
package main
