package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestConcurrency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	inputCh := make(chan Command, 8)
	tickCh := make(chan struct{}, 1)
	carEventsCh := make(chan CarEvent, 32)
	renderCh := make(chan GameState, 1)

	wg.Add(1)
	go runTicker(ctx, &wg, tickCh)

	for _, def := range laneDefs {
		wg.Add(1)
		go runLane(ctx, &wg, def, carEventsCh)
	}

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

	wg.Add(1)
	go runGameLoop(ctx, cancel, &wg, inputCh, tickCh, carEventsCh, renderCh)

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
