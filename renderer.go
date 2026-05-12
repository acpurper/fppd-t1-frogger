package main

import (
	"context"
	"os"
	"strings"
	"sync"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiWhite  = "\x1b[37m"
)

var laneRowDir map[int]Direction

func init() {
	laneRowDir = make(map[int]Direction, len(laneDefs))
	for _, d := range laneDefs {
		laneRowDir[d.Row] = d.Dir
	}
}

func runRenderer(ctx context.Context, wg *sync.WaitGroup, renderCh <-chan GameState) {
	defer wg.Done()
	os.Stdout.WriteString("\x1b[?25l")

	for {
		select {
		case <-ctx.Done():
			return
		case state := <-renderCh:
			renderFrame(state)
		}
	}
}

func renderFrame(state GameState) {
	var sb strings.Builder

	sb.WriteString("\x1b[H\x1b[2J")

	sb.WriteString(ansiBold + "  FROGGER   " + ansiReset)
	sb.WriteString("Lives: ")
	for i := 0; i < state.Lives; i++ {
		sb.WriteString(ansiRed + ansiBold + "♥ " + ansiReset)
	}
	sb.WriteString("\r\n")

	sb.WriteString("+" + strings.Repeat("-", GridCols) + "+\r\n")

	for row := 0; row < GridRows; row++ {
		sb.WriteByte('|')

		chars := make([]byte, GridCols)
		colors := make([]string, GridCols)

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

		if row == state.FrogRow && state.FrogCol >= 0 && state.FrogCol < GridCols {
			chars[state.FrogCol] = '@'
			colors[state.FrogCol] = ansiGreen + ansiBold
		}

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

	sb.WriteString("+" + strings.Repeat("-", GridCols) + "+\r\n")

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
