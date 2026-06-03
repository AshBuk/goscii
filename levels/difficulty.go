// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package levels

// Difficulty controls task complexity for a mission.
type Difficulty string

const (
	Easy     Difficulty = "easy"
	Medium   Difficulty = "medium"
	Hard     Difficulty = "hard"
	Survival Difficulty = "survival" // broken GOSCII voice; no output
)

// Difficulties is the ordered list shown in the difficulty selector.
var Difficulties = []Difficulty{Easy, Medium, Hard, Survival}
