// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/tui"
)

var adventureCmd = &cobra.Command{
	Use:   "adventure [name]",
	Short: "Run a handcrafted offline adventure",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAdventureCmd,
}

func runAdventureCmd(_ *cobra.Command, args []string) error {
	name := "onboarding"
	if len(args) > 0 {
		name = args[0]
	}
	p, err := engine.LoadProgress()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}
	runner := tui.NewAdventureRunner(name, p)
	if runner == nil {
		return fmt.Errorf("adventure %q not found or has no missions", name)
	}
	_, err = tea.NewProgram(runner).Run()
	return err
}
