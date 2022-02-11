// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shutdown

import (
	"context"
	"sync"
)

// Graceful provides a synchronization mechanism to terminate goroutines and
// wait for their termination in a graceful manner.
type Graceful struct {
	wg       sync.WaitGroup
	quit     chan struct{}
	quitOnce sync.Once
}

// NewGraceful creates a new instance of Graceful shutdown.
func NewGraceful() *Graceful {
	return &Graceful{
		quit: make(chan struct{}),
	}
}

// Add adds delta, which may be negative, to the Shutdown WaitGroup counter. If
// the counter becomes zero, all goroutines blocked on Wait are released. If the
// counter goes negative, Add panics.
func (g *Graceful) Add(delta int) {
	g.wg.Add(delta)
}

// Done decrements the Shutdown WaitGroup counter by one.
func (g *Graceful) Done() {
	g.wg.Done()
}

// Quit returns a channel that is closed when the Shutdown method is called.
func (g *Graceful) Quit() <-chan struct{} {
	return g.quit
}

// Shutdown closed the Quit channel and waits for the WaitGroup.
func (g *Graceful) Shutdown(ctx context.Context) error {
	g.quitOnce.Do(func() {
		close(g.quit)
	})
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return nil
}

// Context creates a new context that will be canceled when Graceful is shut
// down.
func (g *Graceful) Context(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-g.Quit():
		case <-ctx.Done():
		}
		cancel()
	}()

	return ctx
}
