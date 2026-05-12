package main

import (
	"context"
	"os"
	"strings"
	"sync"
)

// ANSI helpers
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiWhite  = "\x1b[37m"
)

// laneRowDir maps a row number to its lane direction for rendering.
// Populated once at startup (before any goroutine touches it).
var laneRowDir map[int]Direction

func init() {
	laneRowDir = make(map[int]Direction, len(laneDefs))
	for _, d := range laneDefs {
		laneRowDir[d.Row] = d.Dir
	}
}

// runRenderer receives GameState snapshots and redraws the terminal each time.
func runRenderer(ctx context.Context, wg *sync.WaitGroup, renderCh <-chan GameState) {
	defer wg.Done()
	os.Stdout.WriteString("\x1b[?25l") // hide cursor while playing

	for {
		select {
		case <-ctx.Done():
			return
		case state := <-renderCh:
			renderFrame(state)
		}
	}
}

// renderFrame builds the complete frame in a strings.Builder and writes it in
// a single call to minimise flicker.  All newlines are \r\n for raw-mode compat.
func renderFrame(state GameState) {
	var sb strings.Builder

	// Home + clear screen
	sb.WriteString("\x1b[H\x1b[2J")

	// ── Header ────────────────────────────────────────────────────────────
	sb.WriteString(ansiBold + "  FROGGER   " + ansiReset)
	sb.WriteString("Lives: ")
	for i := 0; i < state.Lives; i++ {
		sb.WriteString(ansiRed + ansiBold + "♥ " + ansiReset)
	}
	sb.WriteString("\r\n")

	// ── Top border ────────────────────────────────────────────────────────
	sb.WriteString("+" + strings.Repeat("-", GridCols) + "+\r\n")

	// ── Rows ──────────────────────────────────────────────────────────────
	for row := 0; row < GridRows; row++ {
		sb.WriteByte('|')

		// cells[col] = (character, colorPrefix)
		chars  := make([]byte, GridCols)
		colors := make([]string, GridCols)

		// Background
		switch {
		case row == GoalRow:
			for i := range chars {
				chars[i] = '~'
			}
			chars[0] = '['
			chars[GridCols-1] = ']'
		case row == StartRow:
			for i := range chars {
				chars[i] = '_'
			}
		default:
			if _, isLane := laneRowDir[row]; isLane {
				for i := range chars {
					chars[i] = '-'
				}
			} else {
				for i := range chars {
					chars[i] = ' '
				}
			}
		}

		// Cars
		for _, car := range state.Cars[row] {
			dir := laneRowDir[row]
			for w := 0; w < CarWidth; w++ {
				col := car.Col + w
				if col < 0 || col >= GridCols {
					continue
				}
				if dir == DirRight {
					if w == 0 {
						chars[col] = '='
					} else {
						chars[col] = '>'
					}
				} else {
					if w == 0 {
						chars[col] = '<'
					} else {
						chars[col] = '='
					}
				}
				colors[col] = ansiYellow
			}
		}

		// Frog (drawn last so it's always visible)
		if row == state.FrogRow && state.FrogCol >= 0 && state.FrogCol < GridCols {
			chars[state.FrogCol] = '@'
			colors[state.FrogCol] = ansiGreen + ansiBold
		}

		// Emit row
		for i, ch := range chars {
			if colors[i] != "" {
				sb.WriteString(colors[i])
				sb.WriteByte(ch)
				sb.WriteString(ansiReset)
			} else {
				sb.WriteByte(ch)
			}
		}
		sb.WriteString("|\r\n")
	}

	// ── Bottom border ─────────────────────────────────────────────────────
	sb.WriteString("+" + strings.Repeat("-", GridCols) + "+\r\n")

	// ── Status line ───────────────────────────────────────────────────────
	switch state.Status {
	case StatusWon:
		sb.WriteString(ansiGreen + ansiBold + "  YOU WIN! Congratulations!   " + ansiReset + "\r\n")
	case StatusLost:
		sb.WriteString(ansiRed + ansiBold + "  GAME OVER! Better luck next time." + ansiReset + "\r\n")
	default:
		sb.WriteString(ansiWhite + "  W/A/S/D: move   Q: quit" + ansiReset + "\r\n")
	}

	os.Stdout.WriteString(sb.String())
}
