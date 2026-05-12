package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestConcurrency runs all goroutines that don't need a real TTY (ticker,
// three lanes, game loop, and a stub renderer) under the race detector for
// 600 ms, exercises input commands, and verifies a clean shutdown.
//
// Run with: go test -race ./...
func TestConcurrency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	inputCh     := make(chan Command, 8)
	tickCh      := make(chan struct{}, 1)
	carEventsCh := make(chan CarEvent, 32)
	renderCh    := make(chan GameState, 1)

	// Goroutine 2: ticker
	wg.Add(1)
	go runTicker(ctx, &wg, tickCh)

	// Goroutines 3-5: lanes
	for _, def := range laneDefs {
		wg.Add(1)
		go runLane(ctx, &wg, def, carEventsCh)
	}

	// Stub renderer: drain renderCh so the game loop never blocks.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-renderCh:
			}
		}
	}()

	// Goroutine 7: game loop
	wg.Add(1)
	go runGameLoop(ctx, cancel, &wg, inputCh, tickCh, carEventsCh, renderCh)

	// Simulate player movement.
	cmds := []Command{CmdUp, CmdRight, CmdUp, CmdLeft, CmdDown, CmdUp, CmdUp,
		CmdRight, CmdUp, CmdLeft, CmdUp}
	go func() {
		for _, cmd := range cmds {
			select {
			case inputCh <- cmd:
			case <-ctx.Done():
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	time.Sleep(600 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		// all goroutines exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("goroutines did not exit within 2s after cancel")
	}
}
