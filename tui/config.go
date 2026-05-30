// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

type configStep int

const (
	configStepSignal configStep = iota
	configStepModel
	configStepAPIKey
)

// ConfigModel is the signal setup screen accessible from the hub.
type ConfigModel struct {
	step          configStep
	signals       []engine.Provider
	models        []string
	signalIdx     int
	modIdx        int
	wiredProvider engine.Provider
	apiKey        textinput.Model
	width         int
}

func NewConfigModel(existing *engine.Config) ConfigModel {
	ti := textinput.New()
	ti.Placeholder = "paste your API key..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	m := ConfigModel{
		signals: []engine.Provider{engine.ProviderGroq, engine.ProviderAnthropic, engine.ProviderOpenAI},
		apiKey:  ti,
	}
	if existing != nil && existing.APIKey != "" {
		for i, s := range m.signals {
			if s == existing.Provider {
				m.signalIdx = i
				break
			}
		}
		m.models = modelsFor(m.signals[m.signalIdx])
		for i, mod := range m.models {
			if mod == existing.Model {
				m.modIdx = i
				break
			}
		}
		m.wiredProvider = existing.Provider
		m.apiKey.SetValue(existing.APIKey)
	}
	return m
}

func (m ConfigModel) Init() tea.Cmd { return nil }

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	if m.step == configStepAPIKey {
		var cmd tea.Cmd
		m.apiKey, cmd = m.apiKey.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m ConfigModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case configStepSignal:
		return m.handleSignalKey(msg)
	case configStepModel:
		return m.handleModelKey(msg)
	case configStepAPIKey:
		return m.handleAPIKeyKey(msg)
	}
	return m, nil
}

func (m ConfigModel) handleSignalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.signalIdx > 0 {
			m.signalIdx--
		}
	case "down", "j":
		if m.signalIdx < len(m.signals)-1 {
			m.signalIdx++
		}
	case "enter":
		m.models = modelsFor(m.signals[m.signalIdx])
		m.modIdx = 0
		if m.signals[m.signalIdx] != m.wiredProvider {
			m.apiKey.SetValue("")
		}
		if len(m.models) == 1 {
			m.step = configStepAPIKey
			return m, m.apiKey.Focus()
		}
		m.step = configStepModel
	case "esc":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

func (m ConfigModel) handleModelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.modIdx > 0 {
			m.modIdx--
		}
	case "down", "j":
		if m.modIdx < len(m.models)-1 {
			m.modIdx++
		}
	case "enter":
		m.step = configStepAPIKey
		return m, m.apiKey.Focus()
	case "esc":
		m.step = configStepSignal
	}
	return m, nil
}

func (m ConfigModel) handleAPIKeyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		key := strings.TrimSpace(m.apiKey.Value())
		if key == "" {
			return m, nil
		}
		cfg := &engine.Config{
			Provider: m.signals[m.signalIdx],
			Model:    m.models[m.modIdx],
			APIKey:   key,
		}
		return m, func() tea.Msg {
			_ = engine.SaveConfig(cfg)
			return BackMsg{}
		}
	case "esc":
		if len(m.models) <= 1 {
			m.step = configStepSignal
		} else {
			m.step = configStepModel
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.apiKey, cmd = m.apiKey.Update(msg)
	return m, cmd
}

func (m ConfigModel) View() tea.View {
	cw := contentWidth(m.width)
	var lines []string
	lines = append(lines,
		centerBlock(styleAccent.Render("GOSCII · AI SIGNAL"), cw),
		"",
		m.renderStepBar(cw),
		"",
		styleMuted.Render("Wire AI signal to load mission protocols."),
		"",
	)

	switch m.step {
	case configStepSignal:
		lines = append(lines, styleText.Render("Select signal:"), "")
		for i, p := range m.signals {
			cursor := "  "
			style := styleMuted
			if i == m.signalIdx {
				cursor = "> "
				style = styleText
			}
			lines = append(lines, style.Render(cursor+string(p)))
		}
		lines = append(lines, "", keyHints(cw, "[↑/↓ k/j] navigate   [enter] select   [esc] back"))

	case configStepModel:
		lines = append(lines, styleText.Render("Select model for "+string(m.signals[m.signalIdx])+":"), "")
		for i, model := range m.models {
			cursor := "  "
			style := styleMuted
			if i == m.modIdx {
				cursor = "> "
				style = styleText
			}
			lines = append(lines, style.Render(cursor+model))
		}
		lines = append(lines, "", keyHints(cw, "[↑/↓ k/j] navigate   [enter] select   [esc] back"))

	case configStepAPIKey:
		lines = append(lines, styleText.Render("Enter API key for "+string(m.signals[m.signalIdx])+":"), "")
		lines = append(lines, m.apiKey.View())
		lines = append(lines, "", keyHints(cw, "[enter] save   [esc] back"))
	}

	return tea.NewView(screenFrame(m.width, strings.Join(lines, "\n")))
}

func (m ConfigModel) renderStepBar(w int) string {
	type stepLabel struct {
		label string
		step  configStep
	}
	steps := [3]stepLabel{
		{"① Signal", configStepSignal},
		{"② Model", configStepModel},
		{"③ Key", configStepAPIKey},
	}
	parts := make([]string, len(steps))
	for i, s := range steps {
		switch {
		case m.step == s.step:
			parts[i] = activeStyle.Padding(0, 1).Render(s.label)
		case s.step < m.step:
			parts[i] = stepDoneStyle.Render(s.label)
		default:
			parts[i] = styleDim.Render(s.label)
		}
	}
	return centerBlock(strings.Join(parts, styleMuted.Render(" → ")), w)
}

func modelsFor(p engine.Provider) []string {
	switch p {
	case engine.ProviderGroq:
		return ai.GroqModels
	case engine.ProviderAnthropic:
		return ai.AnthropicModels
	case engine.ProviderOpenAI:
		return ai.OpenAIModels
	default:
		return nil
	}
}
