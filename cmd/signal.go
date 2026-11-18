// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

var signalCmd = &cobra.Command{
	Use:   "signal",
	Short: "Inspect and measure the wired AI signal",
}

var signalModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List the models the wired key can reach",
	RunE: func(cmd *cobra.Command, _ []string) error {
		signal, err := dialSignal()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		ids, err := signal.Models(ctx)
		if err != nil {
			return err
		}
		cfg, _ := engine.LoadConfig()
		band := ai.Bands[cfg.Provider]
		for _, id := range ids {
			mark := " "
			if band.IsVerified(id) {
				mark = "✓"
			}
			fmt.Printf("%s %s\n", mark, id)
		}
		return nil
	},
}

var bench struct {
	topics   string
	kinds    string
	diffs    string
	n        int
	parallel int
	out      string
}

var signalBenchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Forge a matrix of missions and report how the signal holds up",
	Long: `Forge missions across topics, kinds and difficulties with the wired signal and
report: share passed on the first answer, share rescued by repair, failures,
median forge time, tokens spent and repeated titles. Every mission costs tokens
on your key.`,
	RunE: runBench,
}

func runBench(cmd *cobra.Command, _ []string) error {
	signal, err := dialSignal()
	if err != nil {
		return err
	}
	topics, err := pickTopics(bench.topics)
	if err != nil {
		return err
	}
	var kinds []levels.Kind
	for _, k := range splitList(bench.kinds) {
		kinds = append(kinds, levels.Kind(k))
	}
	var diffs []levels.Difficulty
	for _, d := range splitList(bench.diffs) {
		diffs = append(diffs, levels.Difficulty(d))
	}
	cases := forge.Matrix(topics, kinds, diffs, bench.n)
	fmt.Fprintf(os.Stderr, "bench: %d missions on %s · %s\n", len(cases), signal.Name(), signal.Model())

	done := 0
	rep := forge.Bench(cmd.Context(), signal, cases, bench.parallel, func(r forge.BenchRun) {
		done++
		status := "ok"
		switch {
		case !r.OK:
			status = "FAIL " + r.Err
		case !r.FirstTry():
			status = fmt.Sprintf("ok after %d repairs, %d drafts", r.Repairs, r.Drafts)
		}
		fmt.Fprintf(os.Stderr, "[%d/%d] %s/%s %s %s · %s · %s\n", done, len(cases), r.Topic, r.Beat, r.Kind, r.Difficulty, r.Elapsed.Round(time.Second), status)
	})

	s := rep.Summary
	pct := func(n int) string { return fmt.Sprintf("%d (%.0f%%)", n, 100*float64(n)/float64(max(1, s.Missions))) }
	fmt.Printf("signal          %s · %s\n", rep.Signal, rep.Model)
	fmt.Printf("missions        %d\n", s.Missions)
	fmt.Printf("first try       %s\n", pct(s.FirstTry))
	fmt.Printf("rescued         %s\n", pct(s.Rescued))
	fmt.Printf("failed          %s\n", pct(s.Failed))
	fmt.Printf("median time     %s\n", s.MedianElapsed.Round(time.Second))
	fmt.Printf("tokens          %d in / %d out (%d per mission)\n", s.TokensIn, s.TokensOut, (s.TokensIn+s.TokensOut)/max(1, s.Missions))
	fmt.Printf("repeat titles   %d\n", s.RepeatTitles)

	if bench.out != "" {
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(bench.out, data, 0o644)
	}
	return nil
}

func dialSignal() (ai.Provider, error) {
	cfg, err := engine.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !cfg.Wired() {
		return nil, fmt.Errorf("signal not wired: run `goscii start` and open Signal, or export ANTHROPIC_API_KEY / OPENAI_API_KEY / GROQ_API_KEY")
	}
	return ai.Dial(*cfg)
}

func pickTopics(list string) ([]ai.Topic, error) {
	if list == "" || list == "all" {
		return ai.StdlibTopics, nil
	}
	var out []ai.Topic
	for _, slug := range splitList(list) {
		t, ok := ai.FindTopic(slug)
		if !ok {
			return nil, fmt.Errorf("unknown topic %q", slug)
		}
		out = append(out, t)
	}
	return out, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func init() {
	f := signalBenchCmd.Flags()
	f.StringVar(&bench.topics, "topics", "variables,slices,interfaces,channels", "comma-separated topic slugs, or all")
	f.StringVar(&bench.kinds, "kinds", "repair,fault,decode,refit", "comma-separated mission kinds")
	f.StringVar(&bench.diffs, "difficulties", "easy,hard", "comma-separated difficulties")
	f.IntVar(&bench.n, "n", 1, "missions per topic, kind and difficulty")
	f.IntVar(&bench.parallel, "parallel", 4, "missions forged at once")
	f.StringVar(&bench.out, "json", "", "write the full report to this file")
	signalCmd.AddCommand(signalModelsCmd, signalBenchCmd)
}
