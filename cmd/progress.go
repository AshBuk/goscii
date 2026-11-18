// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
)

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show solved missions by topic",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := engine.LoadProgress()
		if err != nil {
			return err
		}
		topics := completedTopics(p.Topics)
		if len(topics) == 0 && p.AdventureDone == 0 {
			fmt.Println("No progress on record. Run `goscii start` to begin.")
			return nil
		}
		for _, slug := range topics {
			fmt.Printf("%s: %d\n", slug, len(p.Topics[slug].Completed))
		}
		if p.AdventureDone > 0 {
			fmt.Printf("adventure: %d\n", p.AdventureDone)
		}
		fmt.Printf("station integrity: %d%%\n", p.Station.Overall())
		return nil
	},
}

func completedTopics(stats map[string]engine.TopicStat) []string {
	topics := make([]string, 0, len(stats))
	for slug, stat := range stats {
		if len(stat.Completed) > 0 {
			topics = append(topics, slug)
		}
	}
	sort.Strings(topics)
	return topics
}
