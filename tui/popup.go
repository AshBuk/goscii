// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Mission, kind, difficulty and archive are modal popups layered over the live
// hub menu — they are HubModel state (not separate screens), so their update
// and render logic lives here rather than in a child model.
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/forge"
	"github.com/AshBuk/goscii/levels"
)

type hubPopup int

const (
	popupNone hubPopup = iota
	popupDifficulty
	popupMission
	popupWish
	popupKind
	popupArchive
)

// openChannel is the last row of the topic list: practise anything, in your own words.
const openChannel = "Open channel"

// kindChoice is one row of the kind popup.
type kindChoice struct {
	label    string
	brief    string
	kind     levels.Kind
	incident bool
}

var kindChoices = []kindChoice{
	{label: "Mixed", brief: "rotate repair, decode, fault and refit"},
	{label: "Repair", brief: levels.Repair.Brief(), kind: levels.Repair},
	{label: "Fault", brief: levels.Fault.Brief(), kind: levels.Fault},
	{label: "Decode", brief: levels.Decode.Brief(), kind: levels.Decode},
	{label: "Refit", brief: levels.Refit.Brief(), kind: levels.Refit},
	{label: "Incident", brief: "one program, grown across the whole chain", incident: true},
}

// kinds lists the choices for the picked topic: Refit needs a beat with a rule.
func (h HubModel) kinds() []kindChoice {
	if h.pick.Topic.Refittable() {
		return kindChoices
	}
	out := make([]kindChoice, 0, len(kindChoices))
	for _, k := range kindChoices {
		if k.kind != levels.Refit {
			out = append(out, k)
		}
	}
	return out
}

// Selection is what the mission popups hand to the generator.
type Selection struct {
	Topic    ai.Topic
	Kind     levels.Kind
	Incident bool
	Wish     string
}

func newWishInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. worker pools with error handling"
	ti.CharLimit = 200
	return ti
}

func (h HubModel) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		h.diffCursor = max(0, h.diffCursor-1)
	case "down", "j":
		h.diffCursor = min(len(levels.Difficulties)-1, h.diffCursor+1)
	case "enter":
		h.difficulty = levels.Difficulties[h.diffCursor]
		h.popup = popupNone
	case "esc":
		h.popup = popupNone
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) updateMission(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		h.topicCursor = max(0, h.topicCursor-1)
	case "down", "j":
		h.topicCursor = min(len(ai.StdlibTopics), h.topicCursor+1)
	case "enter":
		if h.topicCursor == len(ai.StdlibTopics) {
			h.popup = popupWish
			h.wish = newWishInput()
			return h, h.wish.Focus()
		}
		h.pick = Selection{Topic: ai.StdlibTopics[h.topicCursor]}
		h.popup = popupKind
		h.kindCursor = 0
	case "esc":
		h.popup = popupNone
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) updateWish(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			wish := strings.TrimSpace(h.wish.Value())
			if wish == "" {
				return h, nil
			}
			h.pick = Selection{Topic: ai.OpenTopic(wish), Wish: wish}
			h.popup = popupKind
			h.kindCursor = 0
			return h, nil
		case "esc":
			h.popup = popupMission
			return h, nil
		case "ctrl+c":
			return h, tea.Quit
		}
	}
	var cmd tea.Cmd
	h.wish, cmd = h.wish.Update(msg)
	return h, cmd
}

func (h HubModel) updateKind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		h.kindCursor = max(0, h.kindCursor-1)
	case "down", "j":
		h.kindCursor = min(len(h.kinds())-1, h.kindCursor+1)
	case "enter":
		k := h.kinds()[h.kindCursor]
		sel := h.pick
		sel.Kind, sel.Incident = k.kind, k.incident
		h.popup = popupNone
		return h, func() tea.Msg { return TopicSelectedMsg{Selection: sel} }
	case "esc":
		h.popup = popupMission
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

func (h HubModel) updateArchive(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		h.archiveCursor = max(0, h.archiveCursor-1)
	case "down", "j":
		h.archiveCursor = min(len(h.archive)-1, h.archiveCursor+1)
	case "enter":
		if len(h.archive) == 0 {
			h.popup = popupNone
			return h, nil
		}
		r := h.archive[h.archiveCursor]
		h.popup = popupNone
		return h.setChild(newReplay(r))
	case "esc":
		h.popup = popupNone
	case "ctrl+c":
		return h, tea.Quit
	}
	return h, nil
}

const archiveRows = 12

// popupBox renders the active popup's framed content.
func (h HubModel) popupBox() string {
	var title string
	var rows []string
	hint := "[↑/↓] select   [enter] confirm   [esc] back"
	switch h.popup {
	case popupDifficulty:
		title = "DIFFICULTY"
		for i, d := range levels.Difficulties {
			rows = append(rows, popupRow(string(d), "", i == h.diffCursor))
		}
	case popupMission:
		title = "MISSION"
		for i, t := range ai.StdlibTopics {
			badge := ""
			if n := len(h.progress.Topics[t.Slug].Completed); n > 0 {
				badge = fmt.Sprintf("  [%d]", n)
			}
			rows = append(rows, popupRow(t.Title, badge, i == h.topicCursor))
		}
		rows = append(rows, popupRow(openChannel, "  your own words", h.topicCursor == len(ai.StdlibTopics)))
	case popupWish:
		title = "OPEN CHANNEL"
		rows = append(rows, styleMuted.Render("What should GOSCII drill you on?"), "", h.wish.View())
		hint = "[enter] confirm   [esc] back"
	case popupKind:
		title = "MISSION KIND · " + strings.ToUpper(h.pick.Topic.Title)
		for i, k := range h.kinds() {
			rows = append(rows, popupRow(fmt.Sprintf("%-9s", k.label), "  "+k.brief, i == h.kindCursor))
		}
	case popupArchive:
		title = "ARCHIVE"
		if len(h.archive) == 0 {
			rows = append(rows, styleMuted.Render("No archived missions yet. Forged missions land here."))
		}
		start := max(0, min(h.archiveCursor-archiveRows/2, len(h.archive)-archiveRows))
		for i := start; i < min(len(h.archive), start+archiveRows); i++ {
			r := h.archive[i]
			badge := fmt.Sprintf("  %s · %s · %s", r.Mission.Concept, r.Mission.Kind.Or(), r.Mission.Difficulty)
			if r.Solved {
				badge += " ✓"
			}
			rows = append(rows, popupRow(r.Mission.Title, badge, i == h.archiveCursor))
		}
	}
	lines := append([]string{styleAccent.Render(title), ""}, rows...)
	lines = append(lines, "", styleHint.Render(hint))
	return menuBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// popupRow renders one popup list line: a cursor caret, the name, and an
// optional dim badge.
func popupRow(name, badge string, active bool) string {
	cursor := "  "
	style := styleMuted
	if active {
		cursor = "> "
		style = styleText
	}
	return style.Render(cursor+name) + styleDim.Render(badge)
}

// overlayCenter composites popup centered over base via the lipgloss layer
// compositor (the same mechanism Crush uses for its modals).
func overlayCenter(base, popup string, w, h int) string {
	x := max(0, (w-lipgloss.Width(popup))/2)
	y := max(0, (h-lipgloss.Height(popup))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(popup).X(x).Y(y).Z(1),
	).Render()
}

// replay plays one archived mission offline.
type replay struct {
	cockpit Cockpit
	id      string
}

func newReplay(r forge.Record) tea.Model {
	c := New(r.Mission, r.Template, nil, 0, 0, r.Mission.Concept)
	return &replay{cockpit: c, id: r.Mission.ID}
}

func (r *replay) Init() tea.Cmd { return r.cockpit.Init() }

func (r *replay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case BackMsg, NextMsg:
		id := r.id
		return r, func() tea.Msg {
			if r.cockpit.Passed() {
				forge.MarkSolved(id)
			}
			return BackMsg{}
		}
	}
	m, cmd := r.cockpit.Update(msg)
	r.cockpit = m.(Cockpit)
	return r, cmd
}

func (r *replay) View() tea.View {
	v := r.cockpit.View()
	v.AltScreen = true
	return v
}
