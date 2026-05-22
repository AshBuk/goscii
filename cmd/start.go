// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
	"github.com/AshBuk/goscii/tui"
)

var adventure string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start or continue the campaign",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().StringVar(&adventure, "adventure", "", "start a specific adventure by name")
}

func runStart(_ *cobra.Command, _ []string) error {
	progress, err := engine.LoadProgress()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	var levelPath string
	if adventure != "" {
		levelPath = "adventures/" + adventure
	} else {
		levelPath = progress.CurrentLevel
		if levelPath == "" {
			levelPath = "onboarding/01_variables"
		}
	}

	level, tmpl, err := levels.Load(levelPath)
	if err != nil {
		return fmt.Errorf("load level: %w", err)
	}

	m := tui.New(level, tmpl)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
