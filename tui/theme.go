// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"charm.land/lipgloss/v2"
)

// gosciiLogo keeps its leading/trailing spaces intact — every line is 40 cells
// wide so centerBlock aligns them as one block. Do not TrimSpace it.
const gosciiLogo = ` ██████╗  ██████╗ ███████╗ ██████╗██╗██╗
██╔════╝ ██╔═══██╗██╔════╝██╔════╝██║██║
██║  ███╗██║   ██║███████╗██║     ██║██║
██║   ██║██║   ██║╚════██║██║     ██║██║
╚██████╔╝╚██████╔╝███████║╚██████╗██║██║
 ╚═════╝  ╚═════╝ ╚══════╝ ╚═════╝╚═╝╚═╝`

// Light-purple (lavender) palette.
var (
	colorAccent = lipgloss.Color("141") // light purple — logo, selected item, fills
	colorPulse  = lipgloss.Color("183") // brighter lavender — logo pulse
	colorText   = lipgloss.Color("189") // pale lavender — active items, labels
	colorMuted  = lipgloss.Color("146") // muted lavender — inactive items, separators
	colorDim    = lipgloss.Color("103") // slate purple — values, secondary text
	colorHint   = lipgloss.Color("97")  // purple-grey — key hint bar
	colorWarn   = lipgloss.Color("215") // soft amber — "not wired" warning
	colorPass   = lipgloss.Color("120") // soft mint green — mission passed
	colorFail   = lipgloss.Color("210") // soft coral — mission failed, crash
)

var (
	styleAccent = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	stylePulse  = lipgloss.NewStyle().Bold(true).Foreground(colorPulse)
	styleText   = lipgloss.NewStyle().Foreground(colorText)
	styleMuted  = lipgloss.NewStyle().Foreground(colorMuted)
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleHint   = lipgloss.NewStyle().Foreground(colorHint)
	styleWarn   = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	stylePass   = lipgloss.NewStyle().Foreground(colorPass).Bold(true)
	styleFail   = lipgloss.NewStyle().Foreground(colorFail)
)

// activeStyle is the shared "active = filled accent" treatment, reused for the hub menu and config step indicators.
var (
	activeStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(colorAccent)
	stepDoneStyle = lipgloss.NewStyle().Foreground(colorPass)
)

// menuBoxStyle frames a centered menu list in a rounded accent border.
var menuBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorAccent).
	Padding(0, 3)

// World-view styles — used across world.go and cockpit.go.
var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	gosciiStyle = lipgloss.NewStyle().Foreground(colorHint).Italic(true)
	storyStyle  = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("255"))
)

// contentWidth returns the usable inner width for a screenFrame of the given terminal width.
// screenFrame: Width(w-2) + Padding(1,2) → inner = w-6.
func contentWidth(frameWidth int) int {
	if frameWidth < 40 {
		frameWidth = 82
	}
	return frameWidth - 6
}

// centerBlock centers s horizontally within w columns.
func centerBlock(s string, w int) string {
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
}

// keyHints renders the key hint bar centered within inner width w.
func keyHints(w int, hints string) string {
	return centerBlock(styleHint.Render(hints), w)
}

// screenFrame wraps body in consistent outer padding.
func screenFrame(width int, body string) string {
	if width < 40 {
		width = 82
	}
	return lipgloss.NewStyle().Width(width-2).Padding(1, 2).Render(body)
}
