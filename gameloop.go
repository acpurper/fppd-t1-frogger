package main

import (
	"context"
	"sync"
	"time"
)

// runTicker drives the 30 FPS clock. It uses a non-blocking send so a slow game
// loop causes dropped ticks rather than a blocked ticker goroutine.
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
			default: // game loop is behind; skip this tick
			}
		}
	}
}

// runGameLoop is the single writer of GameState. It multiplexes three event
// sources and emits snapshots to the renderer via a non-blocking send.
//
// State mutation rules:
//   - Only this goroutine reads or writes GameState fields.
//   - Car slices in snapshots are immutable after the channel send (lanes always
//     allocate fresh slices, and here we only replace slice headers, never append).
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

	// endTick != 0 means we are in a post-game countdown before shutdown.
	var endTick uint64

	sendSnapshot := func() {
		// snap copies the [GridRows][]Car array by value (slice headers only).
		// The underlying Car arrays are read-only after being sent here and in lanes.go.
		snap := state
		select {
		case renderCh <- snap:
		default: // renderer busy; it will get the next snapshot
		}
	}

	sendSnapshot() // paint initial frame immediately

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
			// Replace the slice header; never mutate the old backing array.
			state.Cars[evt.Lane] = evt.Cars
			if state.Status == StatusPlaying {
				applyOutcome(&state, &endTick)
			}
			sendSnapshot()
		}
	}
}

// applyOutcome checks for collision and win condition, mutating state as needed.
// endTick is set to schedule a graceful shutdown a few seconds after game-end.
func applyOutcome(state *GameState, endTick *uint64) {
	// Collision: frog occupies a cell covered by a car on the same row.
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
	// Win: frog reached the goal row.
	if state.FrogRow == GoalRow {
		state.Status = StatusWon
		*endTick = state.Tick + 90
	}
}
