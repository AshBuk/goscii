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

// Kind is the shape of a mission: what the player does with the code.
type Kind string

const (
	Repair Kind = "repair" // write the missing code
	Fault  Kind = "fault"  // find and fix a planted bug
	Decode Kind = "decode" // read the program and predict its output
	Refit  Kind = "refit"  // rewrite working code under a construct rule
)

// Kinds is the order shown in the mission popup.
var Kinds = []Kind{Repair, Fault, Decode, Refit}

// Brief is the one-line description shown next to a kind.
func (k Kind) Brief() string {
	switch k {
	case Fault:
		return "find the planted fault"
	case Decode:
		return "predict the output"
	case Refit:
		return "rewrite under a constraint"
	default:
		return "write the missing code"
	}
}

// Or returns k, or Repair when k is empty.
func (k Kind) Or() Kind {
	if k == "" {
		return Repair
	}
	return k
}
