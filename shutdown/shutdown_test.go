// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shutdown_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"resenje.org/x/shutdown"
)

func TestGraceful(t *testing.T) {
	g := shutdown.NewGraceful()

	check := make(chan struct{})
	duration := 200 * time.Millisecond

	g.Add(1)
	go func() {
		defer g.Done()
		defer close(check)
		time.Sleep(duration)
	}()

	start := time.Now()
	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-check:
	case <-time.After(100 * time.Millisecond):
		t.Error("goroutine was not done")
	}

	if time.Since(start) < duration {
		t.Error("shutdown finished before the goroutine is done")
	}
}

func TestGraceful_multipleGoroutines(t *testing.T) {
	g := shutdown.NewGraceful()

	g.Add(1)
	go func() {
		defer g.Done()
		time.Sleep(100 * time.Millisecond)
	}()

	g.Add(1)
	go func() {
		defer g.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	g.Add(1)
	go func() {
		defer g.Done()
		time.Sleep(200 * time.Millisecond)
	}()

	start := time.Now()
	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	if time.Since(start) < 200*time.Millisecond {
		t.Error("shutdown finished before the goroutine is done")
	}
}

func TestGraceful_contextCancelation(t *testing.T) {
	g := shutdown.NewGraceful()

	timeout := 100 * time.Millisecond

	g.Add(1)
	go func() {
		defer g.Done()
		time.Sleep(timeout * 3)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	if err := g.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}

	if time.Since(start) < timeout {
		t.Error("shutdown finished before the timeout")
	}

	if time.Since(start) > timeout*2 {
		t.Error("shutdown was not canceled by context")
	}
}

func TestGraceful_quit(t *testing.T) {
	g := shutdown.NewGraceful()

	check := make(chan struct{})

	g.Add(1)
	go func() {
		defer g.Done()
		defer close(check)

		select {
		case <-g.Quit():
		case <-time.After(time.Minute):
		}
	}()

	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-check:
	case <-time.After(100 * time.Millisecond):
		t.Error("goroutine was not done")
	}
}
