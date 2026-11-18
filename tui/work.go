// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package tui

import (
	"context"
	"sync/atomic"
	"time"
)

var workSeq atomic.Uint64

// nextID returns a token unique for the whole session, so a reply can never
// match a screen or forge it was not started for.
func nextID() uint64 { return workSeq.Add(1) }

// work ties background commands to the screen that started them. Replies carry
// its id and are dropped by any other screen; stop cancels what is in flight.
// It is shared by pointer, so every copy of a bubbletea model sees the same work.
type work struct {
	id      uint64
	cancels []context.CancelFunc
}

func newWork() *work { return &work{id: nextID()} }

// context must be called from Update, never from inside a running command.
func (w *work) context(d time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	w.cancels = append(w.cancels, cancel)
	return ctx, cancel
}

func (w *work) stop() {
	for _, c := range w.cancels {
		c()
	}
	w.cancels = nil
}

// stopper is a screen with background work; the hub stops it before dropping it.
type stopper interface{ stop() }
