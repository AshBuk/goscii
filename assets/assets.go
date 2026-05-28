// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package assets holds the embedded per-topic ASCII art pack.
package assets

import "embed"

//go:embed art
var FS embed.FS
