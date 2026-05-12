package main

import (
	"context"
	"os"
	"sync"
)

// runInput reads raw keypresses from stdin and converts them to Commands.
// A non-WaitGroup sub-goroutine handles the blocking Read so the outer loop
// can exit cleanly on ctx.Done() without being stuck inside a syscall.
func runInput(ctx context.Context, wg *sync.WaitGroup, inputCh chan<- Command) {
	defer wg.Done()

	readCh := make(chan byte, 16)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			// Non-blocking: drop byte if readCh is full.
			select {
			case readCh <- buf[0]:
			default:
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case b := <-readCh:
			if cmd, ok := byteToCommand(b); ok {
				// Non-blocking: drop command if inputCh is full.
				select {
				case inputCh <- cmd:
				default:
				}
			}
		}
	}
}

func byteToCommand(b byte) (Command, bool) {
	switch b {
	case 'w', 'W':
		return CmdUp, true
	case 's', 'S':
		return CmdDown, true
	case 'a', 'A':
		return CmdLeft, true
	case 'd', 'D':
		return CmdRight, true
	case 'q', 'Q', 3: // 3 = Ctrl+C in raw mode
		return CmdQuit, true
	}
	return 0, false
}
