package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

func main() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		os.Stderr.WriteString("frogger: stdin must be an interactive terminal\n")
		os.Exit(1)
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		os.Stderr.WriteString("frogger: cannot set raw mode: " + err.Error() + "\n")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	inputCh := make(chan Command, 8)
	tickCh := make(chan struct{}, 1)
	carEventsCh := make(chan CarEvent, 32)
	renderCh := make(chan GameState, 1)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	wg.Add(1)
	go runInput(ctx, &wg, inputCh)

	wg.Add(1)
	go runTicker(ctx, &wg, tickCh)

	for _, def := range laneDefs {
		wg.Add(1)
		go runLane(ctx, &wg, def, carEventsCh)
	}

	wg.Add(1)
	go runRenderer(ctx, &wg, renderCh)

	wg.Add(1)
	go runGameLoop(ctx, cancel, &wg, inputCh, tickCh, carEventsCh, renderCh)

	wg.Wait()
	signal.Stop(sigCh)

	// Restore terminal to original state.
	_ = term.Restore(fd, oldState)
	os.Stdout.WriteString("\x1b[2J\x1b[H\x1b[?25h") // clear, home, show cursor
}
