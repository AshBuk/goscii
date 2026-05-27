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

	var p *engine.Progress
	signal, err := signalProvider(cfg)
	if err != nil {
		return err
	}

	for {
		p, err = engine.LoadProgress()
		if err != nil {
			return fmt.Errorf("load progress: %w", err)
		}

		e, err := tea.NewProgram(tui.NewEntry(p.Topics), tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		sel := e.(tui.EntryModel).Selected()
		if sel == nil {
			return nil
		}

		ch := ai.NewMissionChain(sel.Topic, sel.Difficulty)
		for !ch.Done() {
			req := ai.Request{
				Topic:      sel.Topic,
				Difficulty: sel.Difficulty,
				Extra:      ch.Brief(),
			}
			m, c, err := runAIMission(signal, req, ch.Step(), ch.Total())
			if err != nil {
				fmt.Println("signal lost:", err)
				break
			}
			if !c.Passed() {
				break
			}
			ch.Record(m.Story, c.PlayerCode())
			p.RecordCompletion(sel.Topic.Slug, m.ID, string(sel.Difficulty))
			if err := engine.SaveProgress(p); err != nil {
				return fmt.Errorf("save progress: %w", err)
			}
			if !c.NextRequested() {
				break
			}
		}
	}
}

func runAIMission(signal ai.Provider, req ai.Request, step, maxLen int) (*levels.Mission, tui.Cockpit, error) {
	fmt.Printf("Wiring %s signal for %q... [%d/%d]\n", req.Difficulty, req.Topic.Title, step, maxLen)

	const maxAttempts = 3
	var (
		m    *levels.Mission
		tmpl string
		err  error
	)
	for attempt := range maxAttempts {
		m, tmpl, err = signal.Generate(context.Background(), req)
		if err == nil && m.Check.Verify(engine.RunCode(tmpl, m.Answer)) {
			break
		}
		if attempt == maxAttempts-1 {
			if err != nil {
				return nil, tui.Cockpit{}, fmt.Errorf("generate mission: %w", err)
			}
			return nil, tui.Cockpit{}, fmt.Errorf("GOSCII signal corrupted after %d attempts: answer does not satisfy check", maxAttempts)
		}
		fmt.Printf("GOSCII signal corrupted. Regenerating... (%d/%d)\n", attempt+1, maxAttempts)
	}

	run, err := tea.NewProgram(tui.New(m, tmpl, signal, step, maxLen), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, tui.Cockpit{}, err
	}
	return m, run.(tui.Cockpit), nil
}

// runAdventure loads and runs an offline handcrafted track level by level.
func runAdventure(name string) error {
	p, err := engine.LoadProgress()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	paths, err := levels.Adventure(name).Missions()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("adventure %q has no missions", name)
	}

	start := 0
	if cp := p.AdventureCheckpoint; cp != "" {
		for i, mp := range paths {
			if mp == cp {
				start = i
				break
			}
		}
	}

	for i := start; i < len(paths); i++ {
		m, tmpl, err := levels.Load(paths[i])
		if err != nil {
			return fmt.Errorf("load mission: %w", err)
		}

		run, err := tea.NewProgram(tui.New(m, tmpl, nil, 0, 0), tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		if !run.(tui.Cockpit).Passed() {
			return nil
		}

		if i+1 < len(paths) {
			p.AdventureCheckpoint = paths[i+1]
		} else {
			p.AdventureCheckpoint = ""
		}
		if err := engine.SaveProgress(p); err != nil {
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
