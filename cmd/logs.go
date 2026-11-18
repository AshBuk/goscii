// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Read the crew logs recovered so far",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := engine.LoadProgress()
		if err != nil {
			return err
		}
		total := len(levels.CrewLogs())
		if len(p.Logs) == 0 {
			fmt.Printf("0 of %d crew logs recovered. Complete a mission chain to decrypt a sector.\n", total)
			return nil
		}
		for _, l := range p.Logs {
			fmt.Printf("▸ %s\n%s\n\n", l.ID, l.Text)
		}
		fmt.Printf("%d of %d crew logs recovered.\n", len(p.Logs), total)
		return nil
	},
}
