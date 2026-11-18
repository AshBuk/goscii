// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

// BackMsg is emitted by any child screen to return to the hub.
type BackMsg struct{}

// TopicSelectedMsg is emitted by the mission popups when the player confirms a mission.
type TopicSelectedMsg struct{ Selection Selection }

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
	itemArchive
	itemStation
	itemProgress
	itemQuit
	hubItemCount
)

// HubModel is the top-level router launched by goscii start.
type HubModel struct {
	popup         hubPopup
	cursor        hubItem
	diffCursor    int
	topicCursor   int
	kindCursor    int
	archiveCursor int
	pick          Selection
	wish          textinput.Model
	archive       []forge.Record
	child         tea.Model
	cfg           *engine.Config
	difficulty    levels.Difficulty
	topic         *ai.Topic
	progress      *engine.Progress
	logoBright    bool
	errMsg        string
	width         int
	height        int
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
	return h.cfg != nil && h.cfg.Wired()
}

func (h HubModel) Init() tea.Cmd { return logoTickCmd() }

func (h HubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case BackMsg:
		h.dropChild()
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
		h.topic = &msg.Selection.Topic
		return h.launchGenerate(msg.Selection)

	case NewMissionMsg:
		h.dropChild()
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
		return h.updateKey(msg)
	}
	if h.popup == popupWish {
		return h.updateWish(msg)
	}
	return h, nil
}

// updateKey routes a key to the open popup, or to the menu.
func (h HubModel) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch h.popup {
	case popupDifficulty:
		return h.updateDiff(msg)
	case popupMission:
		return h.updateMission(msg)
	case popupWish:
		return h.updateWish(msg)
	case popupKind:
		return h.updateKind(msg)
	case popupArchive:
		return h.updateArchive(msg)
	default:
		return h.updateMenu(msg)
	}
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

// dropChild cancels the child's background work before the hub forgets it;
// replies still in flight are dropped by their work id.
func (h *HubModel) dropChild() {
	if s, ok := h.child.(stopper); ok {
		s.stop()
	}
	h.child = nil
}

func (h HubModel) setChild(child tea.Model) (HubModel, tea.Cmd) {
	sized, sizeCmd := child.Update(tea.WindowSizeMsg{Width: h.width, Height: h.height})
	h.child = sized
	return h, tea.Batch(h.child.Init(), sizeCmd)
}

func (h HubModel) launchGenerate(sel Selection) (tea.Model, tea.Cmd) {
	signal, err := ai.Dial(*h.cfg)
	if err != nil {
		h.child = nil
		h.popup = popupNone
		h.errMsg = "Signal error: " + err.Error()
		return h, nil
	}
	h.errMsg = ""
	chain := forge.NewChain(sel.Topic, h.difficulty, sel.Kind)
	chain.Incident = sel.Incident
	chain.Wish = sel.Wish
	return h.setChild(NewGeneratorModel(signal, chain, *h.cfg, h.progress))
}

func (h HubModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h.errMsg = "" // any menu interaction dismisses a stale error
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

	case itemArchive:
		h.archive, _ = forge.Archived()
		h.archiveCursor = 0
		h.popup = popupArchive
		return h, nil

	case itemStation:
		return h.setChild(NewStation(h.progress))

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
	}
	if h.errMsg != "" {
		lines = append(lines, centerBlock(styleWarn.Render("⚠  "+h.errMsg), cw), "")
	}
	lines = append(lines, keyHints(cw, "[↑/↓ k/j] navigate   [enter] select   [ctrl+c] quit"))
	frame := screenFrame(h.width, strings.Join(lines, "\n"))
	if h.height > 0 {
		frame = lipgloss.Place(h.width, h.height, lipgloss.Left, lipgloss.Center, frame)
	}
	return frame
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
		signal = menuRow{name: "Signal", value: string(h.cfg.Provider) + " · " + ai.Bands[h.cfg.Provider].Resolve(h.cfg.Model)}
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
		{name: "Archive", value: "replay offline"},
		{name: "Station", value: fmt.Sprintf("integrity %d%%", h.progress.Station.Overall())},
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

// adventureRunner is a child model that plays through a bundled offline mission track.
type adventureRunner struct {
	paths    []string
	idx      int
	cockpit  Cockpit
	progress *engine.Progress
	credited bool // the current mission's reward is already recorded
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
	if p != nil {
		start = levels.Resume(paths, p.AdventureCheckpoint)
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
	// The hub intercepts BackMsg, so it only reaches the runner when the runner
	// is the root program (`goscii adventure`): there, leaving means quitting.
	if _, ok := msg.(BackMsg); ok {
		r.stop()
		return r, tea.Quit
	}
	if _, ok := msg.(NextMsg); ok {
		if !r.cockpit.Passed() {
			return r, nil
		}
		r.cockpit.stop()
		r.idx++
		r.credited = false
		if r.idx >= len(r.paths) {
			return r, func() tea.Msg { return BackMsg{} }
		}
		m, tmpl, err := levels.Load(r.paths[r.idx])
		if err != nil {
			return r, func() tea.Msg { return BackMsg{} }
		}
		c := New(m, tmpl, nil, 0, 0, m.Concept)
		sized, _ := c.Update(tea.WindowSizeMsg{Width: r.width, Height: r.height})
		r.cockpit = sized.(Cockpit)
		return r, r.cockpit.Init()
	}

	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		r.width = sz.Width
		r.height = sz.Height
	}
	newCockpit, cmd := r.cockpit.Update(msg)
	r.cockpit = newCockpit.(Cockpit)
	return r, tea.Batch(cmd, r.credit())
}

// credit records a passed mission at once and moves the checkpoint past it,
// so leaving before ctrl+n resumes on the next mission.
func (r *adventureRunner) credit() tea.Cmd {
	if r.credited || !r.cockpit.Passed() || r.progress == nil {
		return nil
	}
	r.credited = true
	r.progress.AdventureDone++
	if m := r.cockpit.mission; m.System != "" {
		r.progress.Station.Repair(m.System, repairGain(m.Difficulty))
	}
	r.progress.AdventureCheckpoint = ""
	if r.idx+1 < len(r.paths) {
		r.progress.AdventureCheckpoint = r.paths[r.idx+1]
	}
	return saveProgressCmd(r.progress)
}

func (r *adventureRunner) stop() { r.cockpit.stop() }

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
