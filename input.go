package main

import (
	"context"
	"os"
	"syscall"
	"time"
)

// input.go contém a lógica de leitura do teclado em modo raw. A função
// runInput roda em uma goroutine dedicada e envia comandos para o game loop
// através de `inputCh`.

// runInput é a goroutine responsável por ler o stdin e transformar bytes em comandos.
// O QUE: lê bytes do teclado e converte em valores do tipo Command (CmdUp, CmdDown...).
// POR QUE: usamos um channel (`inputCh`) para enviar comandos ao game loop em vez de
// compartilhar variáveis, porque canais garantem que produtor e consumidor não acessem
// os mesmos dados ao mesmo tempo (evitando data races).
// COMO encerra: a função observa `ctx.Done()` em seu loop; quando `cancel()` é chamado
// (em main ou no gameLoop), runInput retorna e encerra.
func runInput(ctx context.Context, inputCh chan<- Command) {
	// Obtemos o descritor do stdin e colocamos em non-blocking para que a
	// leitura não trave a goroutine quando não houver dados.
	fd := int(os.Stdin.Fd())
	_ = syscall.SetNonblock(fd, true)
	defer syscall.SetNonblock(fd, false)

	buf := make([]byte, 32)

	const (
		escStateNone = iota
		escStateEsc
		escStateEscBracket
		escStateZero
	)
	var state int

	// Loop principal: tenta ler sem bloquear; quando não houver dados, faz
	// uma pequena pausa e re-tenta. Em cada byte lido, converte para Command
	// e envia pelo channel `inputCh`.
	for {
		select {
		case <-ctx.Done():
			// Quando o contexto for cancelado, encerramos a goroutine.
			return
		default:
			n, err := syscall.Read(fd, buf)
			if err != nil {
				if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				return
			}
			if n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			for i := 0; i < n; i++ {
				if cmd, ok := parseInputByte(&state, buf[i]); ok {
					sendCommand(inputCh, cmd)
				}
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
