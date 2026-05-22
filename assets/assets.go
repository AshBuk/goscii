// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

// Package assets holds the embedded astronaut sprite pack.
package assets

import "embed"

//go:embed astronaut
var FS embed.FS
