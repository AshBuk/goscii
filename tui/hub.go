// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/levels"
)

// BackMsg is emitted by any child screen to return to the hub.
type BackMsg struct{}

// TopicSelectedMsg is emitted by the mission popup when the player confirms a topic.
type TopicSelectedMsg struct{ Topic ai.Topic }

// NextMsg is emitted by Cockpit when the player requests the next mission in a chain.
type NextMsg struct{}

// NewMissionMsg is emitted by GeneratorModel's failure screen to reopen the mission popup.
type NewMissionMsg struct{}

type logoTickMsg struct{}

func logoTickCmd() tea.Cmd {
	return tea.Tick(700*time.Millisecond, func(time.Time) tea.Msg { return logoTickMsg{} })
}

type hubItem int

const (
	itemSignal hubItem = iota
	itemMission
	itemAdventure
	itemDifficulty
	itemProgress
	itemQuit
	hubItemCount
)

// HubModel is the top-level router launched by goscii start.
type HubModel struct {
	popup       hubPopup
	cursor      hubItem
	diffCursor  int
	topicCursor int
	child       tea.Model
	cfg         *engine.Config
	difficulty  levels.Difficulty
	topic       *ai.Topic
	progress    *engine.Progress
	logoBright  bool
	width       int
	height      int
}

func NewHub(cfg *engine.Config, p *engine.Progress) HubModel {
	if p == nil {
		p = &engine.Progress{Topics: make(map[string]engine.TopicStat)}
	}
	return HubModel{
		cfg:        cfg,
		difficulty: levels.Easy,
		progress:   p,
		logoBright: true,
	}
}

func (h HubModel) signalWired() bool {
	return h.cfg != nil && h.cfg.APIKey != ""
}

func (h HubModel) Init() tea.Cmd { return logoTickCmd() }

func (h HubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case BackMsg:
		h.child = nil
		h.popup = popupNone
		return h, tea.Batch(reloadHubStateCmd(), logoTickCmd())
	}

	switch msg := msg.(type) {
	case logoTickMsg:
		h.logoBright = !h.logoBright
		if h.child == nil {
			return h, logoTickCmd()
		}
		return h, nil

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
		h.child = nil
		h.popup = popupMission
		h.topicCursor = 0
		return h, logoTickCmd()
	}

	if h.child != nil {
		newChild, cmd := h.child.Update(msg)
		h.child = newChild
		return h, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	case tea.KeyPressMsg:
		switch h.popup {
		case popupDifficulty:
			return h.updateDiff(msg)
		case popupMission:
			return h.updateMission(msg)
		default:
			return h.updateMenu(msg)
		}
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

func (h HubModel) setChild(child tea.Model) (HubModel, tea.Cmd) {
	sized, sizeCmd := child.Update(tea.WindowSizeMsg{Width: h.width, Height: h.height})
	h.child = sized
	return h, tea.Batch(h.child.Init(), sizeCmd)
}

func (h HubModel) launchGenerate(topic ai.Topic) (tea.Model, tea.Cmd) {
	signal, err := buildSignal(h.cfg)
	if err != nil {
		h.child = nil
		return h, nil
	}
	return h.setChild(NewGeneratorModel(signal, topic, h.difficulty, h.progress))
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
		return h.setChild(NewConfigModel(h.cfg))

	case itemMission:
		if !h.signalWired() {
			return h, nil
		}
		h.popup = popupMission
		h.topicCursor = 0
		return h, nil

	case itemDifficulty:
		h.popup = popupDifficulty
		h.diffCursor = diffIndex(h.difficulty)
		return h, nil

	case itemProgress:
		return h.setChild(NewProgress(h.progress))

	case itemAdventure:
		runner := NewAdventureRunner("onboarding", h.progress)
		if runner == nil {
			return h, nil
		}
		return h.setChild(runner)

	case itemQuit:
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) View() tea.View {
	if h.child != nil {
		v := h.child.View()
		v.AltScreen = true // alt screen per-frame; the root model owns it
		return v
	}
	content := h.viewMenu()
	if h.popup != popupNone {
		content = overlayCenter(content, h.popupBox(), h.width, h.height)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (h HubModel) viewMenu() string {
	cw := contentWidth(h.width)

	logoStyle := styleAccent
	if !h.logoBright {
		logoStyle = stylePulse
	}

	rows := h.menuRows()
	// innerW spans the widest row so the selected bar and separator fill the panel.
	innerW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r.plain()); w > innerW {
			innerW = w
		}
	}

	body := make([]string, 0, len(rows)+1)
	for i, r := range rows {
		if hubItem(i) == itemQuit {
			body = append(body, styleMuted.Render(strings.Repeat("─", innerW)))
		}
		body = append(body, h.renderRow(hubItem(i), r, innerW))
	}
	menu := menuBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	lines := []string{
		centerBlock(logoStyle.Render(gosciiLogo), cw),
		centerBlock(styleDim.Render("Go Orbital Survival Coding Interactive Interface"), cw),
		"",
		centerBlock(menu, cw),
		"",
		keyHints(cw, "[↑/↓ k/j] navigate   [enter] select   [ctrl+c] quit"),
	}
	return screenFrame(h.width, strings.Join(lines, "\n"))
}

// menuRow is the content of a hub menu line
type menuRow struct {
	name     string
	value    string
	warn     bool
	disabled bool
}

// plain renders the uncolored row text (with the 2-cell cursor gutter) used for
// width measurement and the selected-bar fill.
func (r menuRow) plain() string {
	s := "  " + fmt.Sprintf("%-10s", r.name)
	if r.value != "" {
		s += "  " + r.value
	}
	return s
}

func (h HubModel) menuRows() []menuRow {
	signal := menuRow{name: "Signal", value: "not wired ⚠", warn: true}
	if h.signalWired() {
		signal = menuRow{name: "Signal", value: string(h.cfg.Provider) + " · " + h.cfg.Model}
	}
	mission := menuRow{name: "Mission", value: "select topic →"}
	switch {
	case !h.signalWired():
		mission = menuRow{name: "Mission", value: "· no signal ·", disabled: true}
	case h.topic != nil:
		mission.value = h.topic.Title
	}
	return []menuRow{
		signal,
		mission,
		{name: "Adventure", value: "onboarding"},
		{name: "Difficulty", value: string(h.difficulty)},
		{name: "Progress", value: "missions log"},
		{name: "Quit"},
	}
}

// renderRow styles one menu line: the active row becomes a filled accent bar,
// inactive rows stay two-tone, disabled dim.
func (h HubModel) renderRow(item hubItem, r menuRow, innerW int) string {
	if h.cursor == item && !r.disabled {
		return activeStyle.Width(innerW).Render("❯" + r.plain()[1:])
	}
	if r.disabled {
		return styleDim.Render(r.plain())
	}
	name := styleMuted.Render("  " + fmt.Sprintf("%-10s", r.name))
	if r.value == "" {
		return name
	}
	valStyle := styleDim
	if r.warn {
		valStyle = styleWarn
	}
	return name + "  " + valStyle.Render(r.value)
}

func diffIndex(d levels.Difficulty) int {
	for i, v := range levels.Difficulties {
		if v == d {
			return i
		}
	}
	return 0
}

// buildSignal creates an ai.Provider from the stored config.
func buildSignal(cfg *engine.Config) (ai.Provider, error) {
	switch cfg.Provider {
	case engine.ProviderGroq:
		return ai.NewGroq(cfg.APIKey, cfg.Model), nil
	case engine.ProviderAnthropic:
		return ai.NewAnthropic(cfg.APIKey, cfg.Model), nil
	case engine.ProviderOpenAI:
		return ai.NewOpenAI(cfg.APIKey, cfg.Model), nil
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
	width    int
	height   int
}

// NewAdventureRunner creates a Bubble Tea model for a named offline mission track
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
	c := New(m, tmpl, nil, 0, 0, m.Concept)
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
		c := New(m, tmpl, nil, 0, 0, m.Concept)
		sized, _ := c.Update(tea.WindowSizeMsg{Width: r.width, Height: r.height})
		r.cockpit = sized.(Cockpit)
		return r, tea.Batch(r.cockpit.Init(), saveProgressCmd(r.progress))
	}

	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		r.width = sz.Width
		r.height = sz.Height
	}
	newCockpit, cmd := r.cockpit.Update(msg)
	r.cockpit = newCockpit.(Cockpit)
	return r, cmd
}

func (r *adventureRunner) View() tea.View {
	v := r.cockpit.View()
	v.AltScreen = true // root model for adventure mode (see HubModel.View)
	return v
}

func saveProgressCmd(p *engine.Progress) tea.Cmd {
	return func() tea.Msg {
		if p != nil {
			_ = engine.SaveProgress(p)
		}
		return nil
	}
}
