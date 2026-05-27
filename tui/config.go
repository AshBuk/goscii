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

// ConfigModel is the first-run setup screen for provider, model, and API key.
type ConfigModel struct {
	step      configStep
	signals   []engine.Provider
	models    []string
	signalIdx int
	modIdx    int
	apiKey    textinput.Model
	result    *engine.Config
	width     int
}

func NewConfigModel() ConfigModel {
	ti := textinput.New()
	ti.Placeholder = "paste your API key..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	return ConfigModel{
		signals: []engine.Provider{
			engine.ProviderGroq,
		},
		apiKey: ti,
	}
}

// Result returns the saved config, or nil if the user quit.
func (m ConfigModel) Result() *engine.Config { return m.result }

func (m ConfigModel) Init() tea.Cmd { return nil }

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
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
		m.step = configStepModel
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
		m.result = &engine.Config{
			Provider: m.signals[m.signalIdx],
			Model:    m.models[m.modIdx],
			APIKey:   key,
		}
		return m, tea.Quit
	case "esc":
		m.step = configStepModel
		return m, nil
	}
	var cmd tea.Cmd
	m.apiKey, cmd = m.apiKey.Update(msg)
	return m, cmd
}

func (m ConfigModel) View() string {
	var lines []string
	lines = append(lines, entryAccent.Render("GOSCII — SETUP"), "")
	lines = append(lines, entryMuted.Render("Brain module offline. Wire AI signal to restore mission protocols."), "")

	switch m.step {
	case configStepSignal:
		lines = append(lines, entryText.Render("Select signal:"), "")
		for i, p := range m.signals {
			cursor := "  "
			style := entryMuted
			if i == m.signalIdx {
				cursor = "> "
				style = entryText
			}
			lines = append(lines, style.Render(cursor+string(p)))
		}
		lines = append(lines, "", entryKeys.Render("[up/down] navigate   [enter] select   [ctrl+c] quit"))

	case configStepModel:
		lines = append(lines, entryText.Render("Select model for "+string(m.signals[m.signalIdx])+":"), "")
		for i, model := range m.models {
			cursor := "  "
			style := entryMuted
			if i == m.modIdx {
				cursor = "> "
				style = entryText
			}
			lines = append(lines, style.Render(cursor+model))
		}
		lines = append(lines, "", entryKeys.Render("[up/down] navigate   [enter] select   [esc] back"))

	case configStepAPIKey:
		lines = append(lines, entryText.Render("Enter API key for "+string(m.signals[m.signalIdx])+":"), "")
		lines = append(lines, m.apiKey.View())
		lines = append(lines, "", entryKeys.Render("[enter] save   [esc] back"))
	}

	return entryFrame(m.width, strings.Join(lines, "\n"))
}

func modelsFor(p engine.Provider) []string {
	switch p {
	case engine.ProviderGroq:
		return ai.GroqModels
	default:
		return nil
	}
}
