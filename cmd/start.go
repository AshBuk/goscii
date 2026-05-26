// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
	"github.com/AshBuk/goscii/tui"
)

var adventure string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Power up. Pick your signal.",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().StringVar(&adventure, "adventure", "", "run a named offline track (e.g. onboarding)")
}

func runStart(_ *cobra.Command, _ []string) error { //nolint:gocyclo // mission orchestration: sequential steps with error guards
	// --- offline track (onboarding / future handcrafted packs) ---
	if adventure != "" {
		return runAdventure(adventure)
	}

	// --- first-run: ensure AI provider is configured ---
	cfg, err := engine.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.APIKey == "" {
		cfg, err = runConfigScreen()
		if err != nil {
			return err
		}
		if cfg == nil {
			return nil // user quit during setup
		}
		if err := engine.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}

	var progress *engine.Progress
	signal, err := signalProvider(cfg)
	if err != nil {
		return err
	}

	for {
		progress, err = engine.LoadProgress()
		if err != nil {
			return fmt.Errorf("load progress: %w", err)
		}

		finalEntry, err := tea.NewProgram(tui.NewEntry(progress.Topics), tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		sel := finalEntry.(tui.EntryModel).Selected()
		if sel == nil {
			return nil
		}

		chain := ai.NewMissionChain(sel.Topic, sel.Difficulty)
		for !chain.Done() {
			req := ai.Request{
				Topic:      sel.Topic,
				Difficulty: sel.Difficulty,
				Extra:      chain.Brief(),
			}
			m, game, err := runAIMission(signal, req, chain.Step(), chain.Total())
			if err != nil {
				fmt.Println("signal lost:", err)
				break
			}
			if !game.Passed() {
				break
			}
			chain.Record(m.Story, game.PlayerCode())
			progress.RecordCompletion(sel.Topic.Slug, m.ID, string(sel.Difficulty))
			if err := engine.SaveProgress(progress); err != nil {
				return fmt.Errorf("save progress: %w", err)
			}
			if !game.NextRequested() {
				break
			}
		}
	}
}

func runAIMission(signal ai.Provider, req ai.Request, step, maxLen int) (*levels.Mission, tui.Model, error) {
	fmt.Printf("Wiring %s signal for %q... [%d/%d]\n", req.Difficulty, req.Topic.Title, step, maxLen)

	const maxAttempts = 3
	var (
		m    *levels.Mission
		tmpl string
		err  error
	)
	for attempt := range maxAttempts {
		m, tmpl, err = signal.Generate(context.Background(), req)
		if err != nil {
			return nil, tui.Model{}, fmt.Errorf("generate mission: %w", err)
		}
		result := engine.RunCode(tmpl, m.Answer)
		if m.Check.Verify(result) {
			break
		}
		if attempt == maxAttempts-1 {
			return nil, tui.Model{}, fmt.Errorf("GOSCII signal corrupted after %d attempts: answer does not satisfy check", maxAttempts)
		}
		fmt.Printf("GOSCII signal corrupted. Regenerating... (%d/%d)\n", attempt+1, maxAttempts)
	}

	finalGame, err := tea.NewProgram(tui.New(m, tmpl, signal, step, maxLen), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, tui.Model{}, err
	}
	return m, finalGame.(tui.Model), nil
}

// runAdventure loads and runs an offline handcrafted track level by level.
func runAdventure(name string) error {
	progress, err := engine.LoadProgress()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	paths, err := levels.Adventure(name).Missions()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("adventure %q has no levels", name)
	}

	start := 0
	if cp := progress.AdventureCheckpoint; cp != "" {
		for i, p := range paths {
			if p == cp {
				start = i
				break
			}
		}
	}

	for i := start; i < len(paths); i++ {
		m, tmpl, err := levels.Load(paths[i])
		if err != nil {
			return fmt.Errorf("load level: %w", err)
		}

		finalGame, err := tea.NewProgram(tui.New(m, tmpl, nil, 0, 0), tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		if !finalGame.(tui.Model).Passed() {
			return nil
		}

		if i+1 < len(paths) {
			progress.AdventureCheckpoint = paths[i+1]
		} else {
			progress.AdventureCheckpoint = ""
		}
		if err := engine.SaveProgress(progress); err != nil {
			return fmt.Errorf("save progress: %w", err)
		}
	}
	return nil
}

// runConfigScreen shows the setup TUI and returns the filled config.
func runConfigScreen() (*engine.Config, error) {
	cm := tui.NewConfigModel()
	final, err := tea.NewProgram(cm, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}
	return final.(tui.ConfigModel).Result(), nil
}

// signalProvider builds the AI provider from the stored signal config.
func signalProvider(cfg *engine.Config) (ai.Provider, error) {
	switch cfg.Provider {
	case engine.ProviderGroq:
		return ai.NewGroq(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("signal %q is not wired yet; reset config and choose groq", cfg.Provider)
	}
}
