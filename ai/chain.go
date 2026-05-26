// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"fmt"
	"strings"

	"github.com/AshBuk/goscii/levels"
)

// ChainEntry records one completed mission in a chain.
type ChainEntry struct {
	Story      string
	PlayerCode string
}

// MissionChain tracks a sequence of related missions for one topic + difficulty.
type MissionChain struct {
	Topic      Topic
	Difficulty levels.Difficulty
	entries    []ChainEntry
	maxLen     int
}

// NewMissionChain starts a new chain for the given topic and difficulty.
func NewMissionChain(topic Topic, d levels.Difficulty) MissionChain {
	return MissionChain{
		Topic:      topic,
		Difficulty: d,
		maxLen:     chainLength(d),
	}
}

func chainLength(d levels.Difficulty) int {
	switch d {
	case levels.Easy:
		return 5
	case levels.Medium:
		return 10
	case levels.Hard, levels.Survival:
		return 20
	default:
		return 5
	}
}

// Record adds a completed mission to the chain history.
func (c *MissionChain) Record(story, playerCode string) {
	c.entries = append(c.entries, ChainEntry{Story: story, PlayerCode: playerCode})
}

// Done reports whether the chain has reached its mission limit.
func (c *MissionChain) Done() bool { return len(c.entries) >= c.maxLen }

// Step returns the current mission number (1-based).
func (c *MissionChain) Step() int { return len(c.entries) + 1 }

// Total returns the total number of missions in this chain.
func (c *MissionChain) Total() int { return c.maxLen }

// Brief formats recent chain history as extra instructions for the next generation.
// Only the last 3 entries are included to avoid overloading the model context.
func (c *MissionChain) Brief() string {
	if len(c.entries) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "This is mission %d of %d in a chain on the same station system.\n", c.Step(), c.maxLen)
	sb.WriteString("Recent missions the player solved:\n")
	start := max(0, len(c.entries)-3)
	for i, e := range c.entries[start:] {
		fmt.Fprintf(&sb, "%d. Story: %s\n   Player wrote:\n%s\n", start+i+1, e.Story, e.PlayerCode)
	}
	sb.WriteString("Keep the same station system and narrative thread. Escalate the Go concept naturally. Do not repeat a concept already covered.")
	return sb.String()
}
