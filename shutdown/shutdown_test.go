// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shutdown_test

import (
	"context"
	"errors"
	"runtime"
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

func TestGraceful_Context(t *testing.T) {
	g := shutdown.NewGraceful()

	duration := 200 * time.Millisecond

	g.Add(1)
	go func() {
		defer g.Done()
		time.Sleep(duration)
	}()

	ctx := g.Context(context.Background())

	start := time.Now()
	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("context was not done")
	}

	if time.Since(start) < duration {
		t.Error("shutdown finished before the context is done")
	}
}

func TestGraceful_Context_backgroundNoLeak(t *testing.T) {
	g := shutdown.NewGraceful()

	startGoroutines := runtime.NumGoroutine()

	var contexts []context.Context
	for i := 0; i < 1000; i++ {
		ctx := g.Context(context.Background())
		if ctx.Done() == nil {
			t.Fatal("expected non-nil Done channel")
		}
		if ctx.Err() != nil {
			t.Fatalf("expected nil Err, got %v", ctx.Err())
		}
		contexts = append(contexts, ctx)
	}

	if leaked := runtime.NumGoroutine() - startGoroutines; leaked > 10 {
		t.Fatalf("goroutines leaked: %d", leaked)
	}

	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, ctx := range contexts {
		select {
		case <-ctx.Done():
		default:
			t.Fatal("expected context to be canceled after shutdown")
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", ctx.Err())
		}
	}
}

func TestGraceful_Context_withoutCancelNoLeak(t *testing.T) {
	g := shutdown.NewGraceful()

	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	detachedCtx := context.WithoutCancel(baseCtx)

	startGoroutines := runtime.NumGoroutine()

	var contexts []context.Context
	for i := 0; i < 1000; i++ {
		ctx := g.Context(detachedCtx)
		contexts = append(contexts, ctx)
	}

	if leaked := runtime.NumGoroutine() - startGoroutines; leaked > 10 {
		t.Fatalf("goroutines leaked: %d", leaked)
	}

	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, ctx := range contexts {
		select {
		case <-ctx.Done():
		default:
			t.Fatal("expected context to be canceled after shutdown")
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", ctx.Err())
		}
	}
}

func TestGraceful_Context_cancellableContext(t *testing.T) {
	g := shutdown.NewGraceful()

	parentCtx, parentCancel := context.WithCancel(context.Background())
	ctx := g.Context(parentCtx)

	select {
	case <-ctx.Done():
		t.Fatal("context should not be done yet")
	default:
	}

	parentCancel()

	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be done after parent canceled")
	}

	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}

func TestGraceful_Context_alreadyShutDown(t *testing.T) {
	g := shutdown.NewGraceful()
	if err := g.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx := g.Context(context.Background())
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context to be immediately done")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}

func TestGraceful_Context_values(t *testing.T) {
	type keyType struct{}
	key := keyType{}
	val := "test-val"

	parent := context.WithValue(context.Background(), key, val)

	g := shutdown.NewGraceful()
	ctx := g.Context(parent)

	if got := ctx.Value(key); got != val {
		t.Fatalf("expected value %q, got %q", val, got)
	}
}
