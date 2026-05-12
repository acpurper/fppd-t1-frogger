package main

import (
	"context"
	"sync"
	"time"
)

// gameloop.go contém o ticker (gerador de frames) e o loop principal do jogo.
// O game loop é o "dono" do estado do jogo (GameState) e faz a maior parte
// da lógica de sincronização entre entrada, física e renderização.

// runTicker gera pulsos de tempo (~30 FPS) e envia sinais pelo channel tickCh.
// O QUE: a cada 1/30s tenta enviar um sinal vazio para `tickCh`.
// POR QUE: separamos o tempo do jogo em uma goroutine para não misturar com
// lógica de gameplay; usamos channel porque o gameLoop consome esses ticks.
// COMO encerra: observa `ctx.Done()` e para imediatamente.
func runTicker(ctx context.Context, wg *sync.WaitGroup, tickCh chan<- struct{}) {
	defer wg.Done()

	t := time.NewTicker(time.Second / 30)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Envio não bloqueante: evita travar o ticker se ninguém estiver
			// lendo no momento.
			select {
			case tickCh <- struct{}{}:
			default:
			}
		}
	}
}

// runGameLoop é a goroutine central do jogo.
// O QUE: recebe eventos (input, tick, car events), atualiza o GameState e
// envia snapshots ao renderer.
// POR QUE: concentramos todo o estado mutável numa única goroutine (dono do
// estado) e comunicamos por canais com as outras goroutines. Isso reduz
// enormemente as chances de data races porque ninguém mais modifica o estado
// diretamente.
// COMO encerra: observa `ctx.Done()` no select; além disso, se o jogo termina
// por vitória/derrota ou o jogador apertar 'q', chamamos `cancel()` para
// sinalizar shutdown ao resto do sistema.
func runGameLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	inputCh <-chan Command,
	tickCh <-chan struct{},
	carEventsCh <-chan CarEvent,
	renderCh chan<- GameState,
) {
	defer wg.Done()

	state := GameState{
		FrogRow: StartRow,
		FrogCol: StartCol,
		Lives:   StartLives,
		Status:  StatusPlaying,
	}

	var endTick uint64

	// sendSnapshot faz uma cópia profunda do estado (deep copy) e tenta
	// enviar ao renderer de forma não bloqueante.
	sendSnapshot := func() {
		var snap GameState
		snap = state
		for r := 0; r < GridRows; r++ {
			if len(state.Cars[r]) > 0 {
				copied := make([]Car, len(state.Cars[r]))
				copy(copied, state.Cars[r])
				snap.Cars[r] = copied
			} else {
				snap.Cars[r] = nil
			}
		}
		select {
		case renderCh <- snap:
		default:
		}
	}

	sendSnapshot()

	for {
		select {
		case <-ctx.Done():
			// ctx.Done() é observado e faz a goroutine terminar.
			return

		case <-tickCh:
			// Recebeu um tick: avança a contagem de frames e aplica lógica.
			state.Tick++
			if endTick > 0 && state.Tick >= endTick {
				cancel()
				return
			}
			if state.Status == StatusPlaying {
				applyOutcome(&state, &endTick)
			}
			sendSnapshot()

		case cmd := <-inputCh:
			// Recebeu comando do jogador. Se for quit, pede shutdown.
			if cmd == CmdQuit {
				cancel()
				return
			}
			if state.Status != StatusPlaying {
				continue
			}
			switch cmd {
			case CmdUp:
				if state.FrogRow > 0 {
					state.FrogRow--
				}
			case CmdDown:
				if state.FrogRow < GridRows-1 {
					state.FrogRow++
				}
			case CmdLeft:
				if state.FrogCol > 0 {
					state.FrogCol--
				}
			case CmdRight:
				if state.FrogCol < GridCols-1 {
					state.FrogCol++
				}
			}
			applyOutcome(&state, &endTick)
			sendSnapshot()

		case evt := <-carEventsCh:
			// Recebeu atualização de uma lane (snapshot de carros). Substitui
			// a linha no estado do jogo pela lista recebida.
			state.Cars[evt.Lane] = evt.Cars
			if state.Status == StatusPlaying {
				applyOutcome(&state, &endTick)
			}
			sendSnapshot()
		}
	}
}

// applyOutcome verifica colisões e condições de vitória/derrota.
// O QUE: testa se o sapo colidiu com algum carro na linha atual ou chegou ao
// objetivo; atualiza vidas e status.
// POR QUE: a lógica é feita dentro do game loop para centralizar o estado.
// COMO encerra: setamos endTick para fazer o jogo esperar ~3s e então o
// runGameLoop aciona cancel().
func applyOutcome(state *GameState, endTick *uint64) {
	for _, car := range state.Cars[state.FrogRow] {
		if state.FrogCol >= car.Col && state.FrogCol < car.Col+CarWidth {
			state.Lives--
			if state.Lives <= 0 {
				state.Status = StatusLost
				*endTick = state.Tick + 90 // ~3 s at 30 FPS
			} else {
				state.FrogRow = StartRow
				state.FrogCol = StartCol
			}
			return
		}
	}
	if state.FrogRow == GoalRow {
		state.Status = StatusWon
		*endTick = state.Tick + 90
	}
}
