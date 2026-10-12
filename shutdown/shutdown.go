// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shutdown

import (
	"context"
	"fmt"
	"sync"
)

// Graceful provides a synchronization mechanism to terminate goroutines and
// wait for their termination in a graceful manner.
type Graceful struct {
	wg       sync.WaitGroup
	quit     chan struct{}
	quitOnce sync.Once

	initOnce sync.Once
	ctx      context.Context
	cancel   context.CancelFunc
}

func (g *Graceful) init() {
	g.initOnce.Do(func() {
		if g.quit == nil {
			g.quit = make(chan struct{})
		}
		g.ctx, g.cancel = context.WithCancel(context.Background())
	})
}

// NewGraceful creates a new instance of Graceful shutdown.
func NewGraceful() *Graceful {
	g := &Graceful{}
	g.init()
	return g
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
	g.init()
	return g.quit
}

// Shutdown closed the Quit channel and waits for the WaitGroup.
func (g *Graceful) Shutdown(ctx context.Context) error {
	g.init()
	g.quitOnce.Do(func() {
		close(g.quit)
		g.cancel()
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
// down or when the provided context is canceled.
func (g *Graceful) Context(ctx context.Context) context.Context {
	g.init()

	select {
	case <-g.quit:
		c, cancel := context.WithCancel(ctx)
		cancel()
		return c
	default:
	}

	if ctx.Done() == nil {
		return &gracefulContext{
			Context: ctx,
			g:       g,
		}
	}

	c, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(g.ctx, cancel)
	context.AfterFunc(c, func() {
		stop()
	})
	return c
}

type gracefulContext struct {
	context.Context
	g *Graceful
}

func (c *gracefulContext) Done() <-chan struct{} {
	return c.g.quit
}

func (c *gracefulContext) Err() error {
	select {
	case <-c.g.quit:
		return context.Canceled
	default:
		return c.Context.Err()
	}
}

func (c *gracefulContext) String() string {
	return fmt.Sprintf("%v.WithGracefulShutdown", c.Context)
}
