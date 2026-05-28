// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// BackMsg is emitted by any child screen to return to the hub.
type BackMsg struct{}

// TopicSelectedMsg is emitted by SelectorModel when the player confirms a topic.
type TopicSelectedMsg struct{ Topic ai.Topic }

// NextMsg is emitted by Cockpit when the player requests the next mission in a chain.
type NextMsg struct{}

// NewMissionMsg is emitted by GeneratorModel's failure screen to open topic select.
type NewMissionMsg struct{}

type hubStep int

const (
	stepHubMenu hubStep = iota
	stepHubDiff
)

type hubItem int

const (
	itemSignal hubItem = iota
	itemMission
	itemDifficulty
	itemAdventure
	itemQuit
	hubItemCount
)

// HubModel is the top-level router launched by goscii start.
type HubModel struct {
	step       hubStep
	cursor     hubItem
	diffCursor int
	child      tea.Model
	cfg        *engine.Config
	difficulty levels.Difficulty
	topic      *ai.Topic
	progress   *engine.Progress
	width      int
}

func NewHub(cfg *engine.Config, p *engine.Progress) HubModel {
	if p == nil {
		p = &engine.Progress{Topics: make(map[string]engine.TopicStat)}
	}
	return HubModel{
		cfg:        cfg,
		difficulty: levels.Easy,
		progress:   p,
	}
}

func (h HubModel) signalWired() bool {
	return h.cfg != nil && h.cfg.APIKey != ""
}

func (h HubModel) Init() tea.Cmd { return nil }

func (h HubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case BackMsg:
		h.child = nil
		h.step = stepHubMenu
		return h, reloadHubStateCmd()
	}

	switch msg := msg.(type) {
	case hubStateMsg:
		if msg.cfg != nil {
			h.cfg = msg.cfg
		}
		if msg.progress != nil {
			h.progress = msg.progress
		}
		return h, nil

	case TopicSelectedMsg:
		h.topic = &msg.Topic
		return h.launchGenerate(msg.Topic)

	case NewMissionMsg:
		h.child = NewSelector(h.progress.Topics)
		return h, h.child.Init()
	}

	if h.child != nil {
		newChild, cmd := h.child.Update(msg)
		h.child = newChild
		return h, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
	case tea.KeyMsg:
		if h.step == stepHubDiff {
			return h.updateDiff(msg)
		}
		return h.updateMenu(msg)
	}
	return h, nil
}

type hubStateMsg struct {
	cfg      *engine.Config
	progress *engine.Progress
}

func reloadHubStateCmd() tea.Cmd {
	return func() tea.Msg {
		cfg, _ := engine.LoadConfig()
		p, _ := engine.LoadProgress()
		return hubStateMsg{cfg: cfg, progress: p}
	}
}

func (h HubModel) launchGenerate(topic ai.Topic) (tea.Model, tea.Cmd) {
	signal, err := buildSignal(h.cfg)
	if err != nil {
		h.child = nil
		return h, nil
	}
	h.child = NewGeneratorModel(signal, topic, h.difficulty, h.progress)
	return h, h.child.Init()
}

func (h HubModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit
	case "up", "k":
		h = h.moveCursor(-1)
	case "down", "j":
		h = h.moveCursor(1)
	case "enter":
		return h.selectItem()
	}
	return h, nil
}

func (h HubModel) moveCursor(dir int) HubModel {
	for {
		next := int(h.cursor) + dir
		if next < 0 || next >= int(hubItemCount) {
			break
		}
		h.cursor = hubItem(next)
		if h.cursor == itemMission && !h.signalWired() {
			continue
		}
		break
	}
	return h
}

func (h HubModel) selectItem() (tea.Model, tea.Cmd) {
	switch h.cursor {
	case itemSignal:
		h.child = NewConfigModel(h.cfg)
		return h, h.child.Init()

	case itemMission:
		if !h.signalWired() {
			return h, nil
		}
		h.child = NewSelector(h.progress.Topics)
		return h, h.child.Init()

	case itemDifficulty:
		h.step = stepHubDiff
		h.diffCursor = diffIndex(h.difficulty)
		return h, nil

	case itemAdventure:
		runner := NewAdventureRunner("onboarding", h.progress)
		if runner == nil {
			return h, nil
		}
		h.child = runner
		return h, runner.Init()

	case itemQuit:
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if h.diffCursor > 0 {
			h.diffCursor--
		}
	case "down", "j":
		if h.diffCursor < len(levels.Difficulties)-1 {
			h.diffCursor++
		}
	case "enter":
		h.difficulty = levels.Difficulties[h.diffCursor]
		h.step = stepHubMenu
	case "esc", "ctrl+c":
		h.step = stepHubMenu
	}
	return h, nil
}

func (h HubModel) View() string {
	if h.child != nil {
		return h.child.View()
	}
	if h.step == stepHubDiff {
		return h.viewDiff()
	}
	return h.viewMenu()
}

func (h HubModel) viewMenu() string {
	var lines []string
	lines = append(lines, selectorAccent.Render("GOSCII · HOME HUB"), "")
	for item := range hubItemCount {
		if item == itemQuit {
			lines = append(lines, selectorMuted.Render("  "+strings.Repeat("─", 26)))
		}
		lines = append(lines, h.renderItem(item))
	}
	lines = append(lines, "", selectorKeys.Render("[↑/↓ or k/j] navigate   [enter] select   [ctrl+c] quit"))
	return selectorFrame(h.width, strings.Join(lines, "\n"))
}

func (h HubModel) renderItem(item hubItem) string {
	active := h.cursor == item
	cursor := "  "
	nameStyle := selectorMuted
	valStyle := selectorDim
	if active {
		cursor = "> "
		nameStyle = selectorText
		valStyle = selectorMuted
	}

	switch item {
	case itemSignal:
		var val string
		if h.signalWired() {
			val = valStyle.Render(string(h.cfg.Provider) + " · " + h.cfg.Model)
		} else {
			val = hubWarnStyle.Render("not wired ⚠")
		}
		return nameStyle.Render(cursor+"Signal") + "   " + val

	case itemMission:
		if !h.signalWired() {
			return selectorDim.Render(cursor+"Mission") + "   " + selectorDim.Render("· no signal · wire AI to load mission protocols ·")
		}
		var val string
		if h.topic != nil {
			val = h.topic.Title
		} else {
			val = "select topic →"
		}
		return nameStyle.Render(cursor+"Mission") + "   " + valStyle.Render(val)

	case itemDifficulty:
		return nameStyle.Render(cursor+"Difficulty") + "   " + valStyle.Render(string(h.difficulty))

	case itemAdventure:
		return nameStyle.Render(cursor+"Adventure") + "   " + valStyle.Render("onboarding")

	case itemQuit:
		return nameStyle.Render(cursor + "Quit")
	}
	return ""
}

func (h HubModel) viewDiff() string {
	var lines []string
	lines = append(lines, selectorAccent.Render("GOSCII · DIFFICULTY"), "")
	for i, d := range levels.Difficulties {
		cursor := "  "
		style := selectorMuted
		if i == h.diffCursor {
			cursor = "> "
			style = selectorText
		}
		lines = append(lines, style.Render(cursor+string(d)))
	}
	lines = append(lines, "", selectorKeys.Render("[↑/↓ or k/j] select   [enter] confirm   [esc] back"))
	return selectorFrame(h.width, strings.Join(lines, "\n"))
}

func diffIndex(d levels.Difficulty) int {
	for i, v := range levels.Difficulties {
		if v == d {
			return i
		}
	}
	return 0
}

var hubWarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

// buildSignal creates an ai.Provider from the stored config.
func buildSignal(cfg *engine.Config) (ai.Provider, error) {
	switch cfg.Provider {
	case engine.ProviderGroq:
		return ai.NewGroq(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("signal %q is not wired yet", cfg.Provider)
	}
}

// adventureRunner is a child model that plays through a handcrafted offline mission track.
type adventureRunner struct {
	paths    []string
	idx      int
	cockpit  Cockpit
	progress *engine.Progress
}

// NewAdventureRunner creates a Bubble Tea model for a named offline mission track.
// Returns nil if the adventure has no missions or cannot be loaded.
func NewAdventureRunner(name string, p *engine.Progress) tea.Model {
	paths, err := levels.Adventure(name).Missions()
	if err != nil || len(paths) == 0 {
		return nil
	}
	start := 0
	if p != nil && p.AdventureCheckpoint != "" {
		for i, path := range paths {
			if path == p.AdventureCheckpoint {
				start = i
				break
			}
		}
	}
	m, tmpl, err := levels.Load(paths[start])
	if err != nil {
		return nil
	}
	c := New(m, tmpl, nil, 0, 0)
	return &adventureRunner{paths: paths, idx: start, cockpit: c, progress: p}
}

func (r *adventureRunner) Init() tea.Cmd { return r.cockpit.Init() }

func (r *adventureRunner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(BackMsg); ok {
		return r, func() tea.Msg { return BackMsg{} }
	}

	if _, ok := msg.(NextMsg); ok {
		r.idx++
		if r.idx >= len(r.paths) {
			if r.progress != nil {
				r.progress.AdventureCheckpoint = ""
			}
			return r, tea.Batch(saveProgressCmd(r.progress), func() tea.Msg { return BackMsg{} })
		}
		if r.progress != nil {
			r.progress.AdventureCheckpoint = r.paths[r.idx]
		}
		m, tmpl, err := levels.Load(r.paths[r.idx])
		if err != nil {
			return r, func() tea.Msg { return BackMsg{} }
		}
		r.cockpit = New(m, tmpl, nil, 0, 0)
		return r, tea.Batch(r.cockpit.Init(), saveProgressCmd(r.progress))
	}

	newCockpit, cmd := r.cockpit.Update(msg)
	r.cockpit = newCockpit.(Cockpit)
	return r, cmd
}

func (r *adventureRunner) View() string { return r.cockpit.View() }

func saveProgressCmd(p *engine.Progress) tea.Cmd {
	return func() tea.Msg {
		if p != nil {
			_ = engine.SaveProgress(p)
		}
		return nil
	}
}
