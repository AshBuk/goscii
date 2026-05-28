// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
	step      configStep
	signals   []engine.Provider
	models    []string
	signalIdx int
	modIdx    int
	apiKey    textinput.Model
	width     int
}

func NewConfigModel(existing *engine.Config) ConfigModel {
	ti := textinput.New()
	ti.Placeholder = "paste your API key..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	m := ConfigModel{
		signals: []engine.Provider{engine.ProviderGroq},
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
		m.apiKey.SetValue(existing.APIKey)
	}
	return m
}

func (m ConfigModel) Init() tea.Cmd { return nil }

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, func() tea.Msg { return BackMsg{} }
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

func (m ConfigModel) View() string {
	var lines []string
	lines = append(lines, selectorAccent.Render("GOSCII · AI SIGNAL"), "")
	lines = append(lines, selectorMuted.Render("Brain module offline. Wire AI signal to restore mission protocols."), "")

	switch m.step {
	case configStepSignal:
		lines = append(lines, selectorText.Render("Select signal:"), "")
		for i, p := range m.signals {
			cursor := "  "
			style := selectorMuted
			if i == m.signalIdx {
				cursor = "> "
				style = selectorText
			}
			lines = append(lines, style.Render(cursor+string(p)))
		}
		lines = append(lines, "", selectorKeys.Render("[up/down] navigate   [enter] select   [ctrl+c] back"))

	case configStepModel:
		lines = append(lines, selectorText.Render("Select model for "+string(m.signals[m.signalIdx])+":"), "")
		for i, model := range m.models {
			cursor := "  "
			style := selectorMuted
			if i == m.modIdx {
				cursor = "> "
				style = selectorText
			}
			lines = append(lines, style.Render(cursor+model))
		}
		lines = append(lines, "", selectorKeys.Render("[up/down] navigate   [enter] select   [esc] back"))

	case configStepAPIKey:
		lines = append(lines, selectorText.Render("Enter API key for "+string(m.signals[m.signalIdx])+":"), "")
		lines = append(lines, m.apiKey.View())
		lines = append(lines, "", selectorKeys.Render("[enter] save   [esc] back"))
	}

	return selectorFrame(m.width, strings.Join(lines, "\n"))
}

func modelsFor(p engine.Provider) []string {
	switch p {
	case engine.ProviderGroq:
		return ai.GroqModels
	default:
		return nil
	}
}
