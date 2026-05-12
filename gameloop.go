package main

import (
	"context"
	"sync"
	"time"
)

func runTicker(ctx context.Context, wg *sync.WaitGroup, tickCh chan<- struct{}) {
	defer wg.Done()

	t := time.NewTicker(time.Second / 30)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			select {
			case tickCh <- struct{}{}:
			default:
			}
		}
	}
}

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

	sendSnapshot := func() {

		snap := state
		select {
		case renderCh <- snap:
		default:
		}
	}

	sendSnapshot()

	for {
		select {
		case <-ctx.Done():
			return

		case <-tickCh:
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
			state.Cars[evt.Lane] = evt.Cars
			if state.Status == StatusPlaying {
				applyOutcome(&state, &endTick)
			}
			sendSnapshot()
		}
	}
}

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
