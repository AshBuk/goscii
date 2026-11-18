// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

var forgeOpts struct {
	topic string
	kind  string
	diff  string
	count int
	out   string
}

var forgeCmd = &cobra.Command{
	Use:   "forge <track>",
	Short: "Forge a draft adventure track for hand editing",
	Long: `Forge a chain of verified missions with the wired signal and write them as an
adventure track (level.yaml + template.txt per mission). The result is a draft:
edit the stories, register the track in levels/loader.go, and play it through.
See docs/ADVENTURE_PROTOCOL.md.`,
	Args: cobra.ExactArgs(1),
	RunE: runForge,
}

func runForge(cmd *cobra.Command, args []string) error {
	track := args[0]
	signal, err := dialSignal()
	if err != nil {
		return err
	}
	topic, ok := ai.FindTopic(forgeOpts.topic)
	if !ok {
		return fmt.Errorf("unknown topic %q", forgeOpts.topic)
	}
	out := forgeOpts.out
	if out == "" {
		out = filepath.Join("levels", "adventures", track)
	}
	chain := forge.NewChain(topic, levels.Difficulty(forgeOpts.diff), levels.Kind(forgeOpts.kind))
	p := &engine.Progress{}
	for i := 1; i <= forgeOpts.count; i++ {
		o := chain.Next(p)
		fmt.Fprintf(os.Stderr, "[%d/%d] %s · %s · %s\n", i, forgeOpts.count, o.Beat.Slug, o.Kind, o.System.Name)
		res, err := forge.Forge(cmd.Context(), signal, o, func(e forge.Event) {
			if e.Stage == forge.Recalibrating {
				fmt.Fprintf(os.Stderr, "      recalibrating: %s\n", e.Defect)
			}
		})
		if err != nil {
			return err
		}
		m := res.Mission
		m.ID = fmt.Sprintf("%s_%02d_%s", track, i, m.Beat)
		dir := filepath.Join(out, fmt.Sprintf("%02d_%s", i, strings.ReplaceAll(m.Beat, "-", "_")))
		if err := writeMission(dir, m, res.Template); err != nil {
			return err
		}
		chain.Record(forge.Entry{Title: m.Title, Story: m.Story, Beat: m.Beat, System: m.System, PlayerCode: m.Answer})
		fmt.Fprintf(os.Stderr, "      wrote %s (%q)\n", dir, m.Title)
	}
	fmt.Fprintf(os.Stderr, "Register the track in levels/loader.go:  const X Adventure = %q\n", track)
	return nil
}

const licenseHeader = "# Copyright 2026 Asher Buk\n# SPDX-License-Identifier: Apache-2.0\n# https://github.com/AshBuk/goscii\n\n"

func writeMission(dir string, m *levels.Mission, tmpl string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "level.yaml"), append([]byte(licenseHeader), data...), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "template.txt"), []byte(strings.TrimRight(tmpl, "\n")+"\n"), 0o644)
}

func init() {
	f := forgeCmd.Flags()
	f.StringVar(&forgeOpts.topic, "topic", "variables", "topic slug")
	f.StringVar(&forgeOpts.kind, "kind", "", "mission kind (repair, fault, decode, refit); empty rotates")
	f.StringVar(&forgeOpts.diff, "difficulty", "easy", "difficulty")
	f.IntVar(&forgeOpts.count, "count", 3, "missions to forge")
	f.StringVar(&forgeOpts.out, "out", "", "output directory (default levels/adventures/<track>)")
}
