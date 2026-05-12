package main

import (
	"context"
	"os"
	"strings"
	"sync"
)

// renderer.go contém a lógica de apresentação no terminal (ANSI). O renderer
// consome snapshots do estado do jogo e desenha a tela.

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

// runRenderer recebe snapshots do game loop e imprime no terminal.
// O QUE: desenha o estado recebido em `renderCh` usando códigos ANSI.
// POR QUE: mantemos rendering separado do game logic; o renderer só consome
// dados imutáveis (snapshots) e não altera estado.
// COMO encerra: observa `ctx.Done()` e retorna quando o contexto é cancelado.
func runRenderer(ctx context.Context, wg *sync.WaitGroup, renderCh <-chan GameState) {
	defer wg.Done()

	os.Stdout.WriteString("\x1b[?25l") // esconde cursor

	for {
		select {
		case <-ctx.Done():
			return
		case state := <-renderCh:
			// Recebemos um snapshot deep-copiado pelo game loop, logo é
			// seguro acessar sem riscos de race.
			renderFrame(state)
		}
	}
}

// renderFrame monta a string da tela e escreve no stdout.
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
