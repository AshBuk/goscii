// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/tui"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Power up. Pick your signal.",
	RunE:  runStart,
}

func runStart(_ *cobra.Command, _ []string) error {
	cfg, err := engine.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	p, err := engine.LoadProgress()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}
	_, err = tea.NewProgram(tui.NewHub(cfg, p), tea.WithAltScreen()).Run()
	return err
}
