package main

import (
	"context"
	"os"
	"sync"
)

func runInput(ctx context.Context, wg *sync.WaitGroup, inputCh chan<- Command) {
	defer wg.Done()

	readCh := make(chan byte, 64)
	go func() {
		buf := make([]byte, 16)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			for i := 0; i < n; i++ {
				select {
				case readCh <- buf[i]:
				default:
				}
			}
		}
	}()

	const (
		escStateNone = iota
		escStateEsc
		escStateEscBracket
		escStateZero
	)
	var state int

	for {
		select {
		case <-ctx.Done():
			return
		case b := <-readCh:
			if cmd, ok := parseInputByte(&state, b); ok {
				sendCommand(inputCh, cmd)
			}
		}
	}
}

func parseInputByte(state *int, b byte) (Command, bool) {
	switch *state {
	case 0:
		switch b {
		case 0x1b:
			*state = 1
			return 0, false
		case 0x00, 0xe0:
			*state = 3
			return 0, false
		default:
			return byteToCommand(b)
		}
	case 1:
		if b == '[' || b == 'O' {
			*state = 2
			return 0, false
		}
		*state = 0
		return 0, false
	case 2:
		*state = 0
		switch b {
		case 'A':
			return CmdUp, true
		case 'B':
			return CmdDown, true
		case 'C':
			return CmdRight, true
		case 'D':
			return CmdLeft, true
		}
		return 0, false
	case 3:
		*state = 0
		switch b {
		case 72:
			return CmdUp, true
		case 80:
			return CmdDown, true
		case 77:
			return CmdRight, true
		case 75:
			return CmdLeft, true
		}
		return 0, false
	}
	return 0, false
}

func sendCommand(inputCh chan<- Command, cmd Command) {
	select {
	case inputCh <- cmd:
	default:
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
	case 'q', 'Q', 3:
		return CmdQuit, true
	}
	return 0, false
}
