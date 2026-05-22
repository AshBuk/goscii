// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
)

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show completed levels",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := engine.LoadProgress()
		if err != nil {
			return err
		}
		if p.CurrentLevel == "" {
			fmt.Println("No progress yet. Run `goscii start` to begin.")
			return nil
		}
		fmt.Printf("Current level: %s\n", p.CurrentLevel)
		return nil
	},
}
