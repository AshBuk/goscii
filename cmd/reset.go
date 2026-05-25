// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
)

var resetAll bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset adventure checkpoint or all progress",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := engine.LoadProgress()
		if err != nil {
			return err
		}
		if resetAll {
			p = &engine.Progress{
				Topics: make(map[string]engine.TopicStat),
			}
			fmt.Println("All progress reset.")
		} else {
			p.AdventureCheckpoint = ""
			fmt.Println("Adventure checkpoint reset.")
		}
		return engine.SaveProgress(p)
	},
}

func init() {
	resetCmd.Flags().BoolVar(&resetAll, "all", false, "reset all progress")
}
