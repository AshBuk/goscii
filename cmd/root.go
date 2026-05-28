// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package cmd is the command bridge - parses mission directives from the terminal.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goscii",
	Short: "A terminal game for learning Go",
	Long: `GOSCII - learn Go by helping an astronaut navigate through space.
Write real Go code to solve each mission.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(adventureCmd)
	rootCmd.AddCommand(progressCmd)
	rootCmd.AddCommand(resetCmd)
}
