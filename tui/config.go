// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/AshBuk/goscii/ai"
	"github.com/AshBuk/goscii/engine"
)

type configStep int

const (
	configStepSignal configStep = iota
	configStepEndpoint
	configStepAPIKey
	configStepScanning
	configStepModel
	configStepModelID
)

// otherModel is the last row of the model list: type an ID by hand.
const otherModel = "› other model ID…"

const modelRows = 12 // visible rows in the model list

type scanDoneMsg struct {
	models []string
	err    error
}

// ConfigModel is the signal setup screen accessible from the hub.
type ConfigModel struct {
	step     configStep
	signals  []engine.Provider
	sigIdx   int
	models   []string
	modIdx   int
	scanWarn string
	keyErr   string
	existing engine.Config
	endpoint textinput.Model
	apiKey   textinput.Model
	modelID  textinput.Model
	width    int
}

func NewConfigModel(existing *engine.Config) ConfigModel {
	key := textinput.New()
	key.Placeholder = "paste your API key..."
	key.EchoMode = textinput.EchoPassword
	key.EchoCharacter = '•'

	ep := textinput.New()
	ep.Placeholder = "http://localhost:11434/v1"

	id := textinput.New()
	id.Placeholder = "model ID, e.g. qwen3:32b"

	m := ConfigModel{signals: engine.Providers, apiKey: key, endpoint: ep, modelID: id}
	if existing != nil {
		m.existing = *existing
		if i := slices.Index(m.signals, existing.Provider); i >= 0 {
			m.sigIdx = i
		}
		m.apiKey.SetValue(existing.APIKey)
		m.endpoint.SetValue(existing.BaseURL)
	}
	return m
}

func (m ConfigModel) signal() engine.Provider { return m.signals[m.sigIdx] }

// draft is the config as entered so far.
func (m ConfigModel) draft() engine.Config {
	return engine.Config{
		Provider: m.signal(),
		APIKey:   strings.TrimSpace(m.apiKey.Value()),
		BaseURL:  strings.TrimSpace(m.endpoint.Value()),
		Prefetch: m.existing.Prefetch,
	}
}

func (m ConfigModel) Init() tea.Cmd { return nil }

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case scanDoneMsg:
		return m.scanned(msg)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	return m.updateInput(msg)
}

func (m ConfigModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.step {
	case configStepEndpoint:
		m.endpoint, cmd = m.endpoint.Update(msg)
	case configStepAPIKey:
		m.apiKey, cmd = m.apiKey.Update(msg)
	case configStepModelID:
		m.modelID, cmd = m.modelID.Update(msg)
	}
	return m, cmd
}

func (m ConfigModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case configStepSignal:
		return m.handleSignalKey(msg)
	case configStepEndpoint:
		return m.handleEndpointKey(msg)
	case configStepAPIKey:
		return m.handleAPIKeyKey(msg)
	case configStepScanning:
		if msg.String() == "esc" {
			m.step = configStepAPIKey
		}
		return m, nil
	case configStepModel:
		return m.handleModelKey(msg)
	case configStepModelID:
		return m.handleModelIDKey(msg)
	}
	return m, nil
}

func (m ConfigModel) handleSignalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.sigIdx = max(0, m.sigIdx-1)
	case "down", "j":
		m.sigIdx = min(len(m.signals)-1, m.sigIdx+1)
	case "enter":
		if m.signal() != m.existing.Provider {
			m.apiKey.SetValue("")
		}
		m.keyErr = ""
		if m.signal() == engine.ProviderCustom {
			m.step = configStepEndpoint
			return m, m.endpoint.Focus()
		}
		m.step = configStepAPIKey
		return m, m.apiKey.Focus()
	case "esc":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

func (m ConfigModel) handleEndpointKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if strings.TrimSpace(m.endpoint.Value()) == "" {
			return m, nil
		}
		m.step = configStepAPIKey
		return m, m.apiKey.Focus()
	case "esc":
		m.step = configStepSignal
		return m, nil
	}
	return m.updateInput(msg)
}

func (m ConfigModel) handleAPIKeyKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cfg := m.draft()
		if cfg.Key() == "" && m.signal() != engine.ProviderCustom {
			m.keyErr = "a key is required for this signal"
			return m, nil
		}
		m.keyErr = ""
		m.step = configStepScanning
		return m, scanCmd(cfg)
	case "esc":
		if m.signal() == engine.ProviderCustom {
			m.step = configStepEndpoint
		} else {
			m.step = configStepSignal
		}
		return m, nil
	}
	return m.updateInput(msg)
}

// scanCmd lists the models the key can reach. It is also the key check.
func scanCmd(cfg engine.Config) tea.Cmd {
	return func() tea.Msg {
		cfg.Model = "scan" // Dial needs a model; listing ignores it
		sig, err := ai.Dial(cfg)
		if err != nil {
			return scanDoneMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ids, err := sig.Models(ctx)
		return scanDoneMsg{models: ids, err: err}
	}
}

func (m ConfigModel) scanned(msg scanDoneMsg) (tea.Model, tea.Cmd) {
	if m.step != configStepScanning {
		return m, nil
	}
	if errors.Is(msg.err, ai.ErrKeyRejected) {
		m.step = configStepAPIKey
		m.keyErr = msg.err.Error()
		return m, m.apiKey.Focus()
	}
	m.scanWarn = ""
	if msg.err != nil {
		m.scanWarn = "could not list models: " + msg.err.Error()
	}
	m.models = modelChoices(ai.Bands[m.signal()], msg.models)
	m.modIdx = 0
	if i := slices.Index(m.models, ai.Bands[m.signal()].Resolve(m.existing.Model)); i >= 0 && m.signal() == m.existing.Provider {
		m.modIdx = i
	}
	m.step = configStepModel
	return m, nil
}

// noise filters model IDs that cannot write missions: embeddings, audio, images.
var noise = []string{"embed", "whisper", "tts", "dall-e", "image", "audio", "realtime", "moderation", "guard", "transcribe", "search", "orpheus", "safeguard"}

// modelChoices puts verified models first, then the rest of the live list, then
// the manual entry row.
func modelChoices(b ai.Band, live []string) []string {
	out := slices.Clone(b.Verified)
	rest := []string{}
	for _, id := range live {
		if slices.Contains(out, id) || slices.ContainsFunc(noise, func(n string) bool { return strings.Contains(id, n) }) {
			continue
		}
		rest = append(rest, id)
	}
	slices.Sort(rest)
	return append(append(out, rest...), otherModel)
}

func (m ConfigModel) handleModelKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.modIdx = max(0, m.modIdx-1)
	case "down", "j":
		m.modIdx = min(len(m.models)-1, m.modIdx+1)
	case "enter":
		if m.models[m.modIdx] == otherModel {
			m.step = configStepModelID
			return m, m.modelID.Focus()
		}
		return m, m.save(m.models[m.modIdx])
	case "esc":
		m.step = configStepAPIKey
		return m, m.apiKey.Focus()
	}
	return m, nil
}

func (m ConfigModel) handleModelIDKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if id := strings.TrimSpace(m.modelID.Value()); id != "" {
			return m, m.save(id)
		}
		return m, nil
	case "esc":
		m.step = configStepModel
		return m, nil
	}
	return m.updateInput(msg)
}

func (m ConfigModel) save(model string) tea.Cmd {
	cfg := m.draft()
	cfg.Model = model
	return tea.Sequence(
		func() tea.Msg { return saveFailed("Signal", engine.SaveConfig(&cfg)) },
		func() tea.Msg { return BackMsg{} },
	)
}

func (m ConfigModel) View() tea.View {
	cw := contentWidth(m.width)
	lines := []string{
		centerBlock(styleAccent.Render("GOSCII · AI SIGNAL"), cw),
		"",
		m.renderStepBar(cw),
		"",
		styleMuted.Render("Wire AI signal to load mission protocols."),
		"",
	}
	nav := "[↑/↓ k/j] navigate   [enter] select   [esc] back"

	switch m.step {
	case configStepSignal:
		lines = append(lines, styleText.Render("Select signal:"), "")
		for i, p := range m.signals {
			lines = append(lines, listRow(string(p)+signalNote(p), i == m.sigIdx))
		}
		lines = append(lines, "", keyHints(cw, nav))

	case configStepEndpoint:
		lines = append(lines,
			styleText.Render("Endpoint of an OpenAI-compatible server (Ollama, LM Studio, OpenRouter):"), "",
			m.endpoint.View(), "", keyHints(cw, "[enter] next   [esc] back"))

	case configStepAPIKey:
		lines = append(lines, styleText.Render("Enter API key for "+string(m.signal())+":"), "", m.apiKey.View())
		if src := m.draft().KeySource(); src != "" && src != "config" {
			lines = append(lines, "", styleDim.Render("Leave empty to use $"+src+"."))
		} else if m.signal() == engine.ProviderCustom {
			lines = append(lines, "", styleDim.Render("Leave empty if the server needs no key."))
		}
		if m.keyErr != "" {
			lines = append(lines, "", styleWarn.Render("⚠  "+m.keyErr))
		}
		lines = append(lines, "", keyHints(cw, "[enter] scan band   [esc] back"))

	case configStepScanning:
		lines = append(lines, styleText.Render("Scanning band for reachable models..."), "", keyHints(cw, "[esc] back"))

	case configStepModel:
		lines = append(lines, styleText.Render("Select model for "+string(m.signal())+":"), "")
		band := ai.Bands[m.signal()]
		start := max(0, min(m.modIdx-modelRows/2, len(m.models)-modelRows))
		for i := start; i < min(len(m.models), start+modelRows); i++ {
			id := m.models[i]
			label := id
			if band.IsVerified(id) {
				label += styleDim.Render("  ✓ verified")
			}
			lines = append(lines, listRow(label, i == m.modIdx))
		}
		if m.scanWarn != "" {
			lines = append(lines, "", styleWarn.Render("⚠  "+m.scanWarn))
		}
		lines = append(lines, "", keyHints(cw, nav))

	case configStepModelID:
		lines = append(lines, styleText.Render("Model ID:"), "", m.modelID.View(), "", keyHints(cw, "[enter] save   [esc] back"))
	}
	return tea.NewView(screenFrame(m.width, strings.Join(lines, "\n")))
}

func signalNote(p engine.Provider) string {
	if p == engine.ProviderCustom {
		return styleDim.Render("  any OpenAI-compatible endpoint")
	}
	if engine.EnvKey(p) != "" {
		return styleDim.Render("  key found in environment")
	}
	return ""
}

func listRow(label string, active bool) string {
	if active {
		return styleText.Render("> ") + styleText.Render(label)
	}
	return styleMuted.Render("  " + label)
}

func (m ConfigModel) renderStepBar(w int) string {
	type stepLabel struct {
		label string
		step  configStep
	}
	steps := []stepLabel{{"Signal", configStepSignal}}
	if m.signal() == engine.ProviderCustom {
		steps = append(steps, stepLabel{"Endpoint", configStepEndpoint})
	}
	steps = append(steps, stepLabel{"Key", configStepAPIKey}, stepLabel{"Model", configStepModel})
	cur := m.step
	if cur == configStepScanning {
		cur = configStepAPIKey
	}
	if cur == configStepModelID {
		cur = configStepModel
	}
	parts := make([]string, len(steps))
	for i, s := range steps {
		label := fmt.Sprintf("%d %s", i+1, s.label)
		switch {
		case cur == s.step:
			parts[i] = activeStyle.Padding(0, 1).Render(label)
		case s.step < cur:
			parts[i] = stepDoneStyle.Render(label)
		default:
			parts[i] = styleDim.Render(label)
		}
	}
	return centerBlock(strings.Join(parts, styleMuted.Render(" → ")), w)
}
