package main

import (
	"context"
	"sync"
	"time"
)

// runLane manages one traffic lane. It has its own ticker with a distinct speed,
// moves cars each tick, spawns new ones when the lane is clear enough, and sends
// a complete immutable snapshot to carEventsCh after every update.
//
// Invariant: a new []Car backing array is allocated every tick so the snapshot
// handed to the game loop is never aliased by the lane goroutine.
func runLane(ctx context.Context, wg *sync.WaitGroup, def LaneDef, carEventsCh chan<- CarEvent) {
	defer wg.Done()

	ticker := time.NewTicker(def.Interval)
	defer ticker.Stop()

	var cars []Car // lane owns this slice exclusively between ticks

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate a fresh slice so the old backing array (possibly held by
			// a GameState snapshot in the game loop or renderer) is never mutated.
			var moved []Car
			for _, c := range cars {
				nc := c
				if def.Dir == DirRight {
					nc.Col++
				} else {
					nc.Col--
				}
				// Keep car if any part of it is still within or entering the grid.
				if nc.Col < GridCols && nc.Col+CarWidth > 0 {
					moved = append(moved, nc)
				}
			}
			cars = moved

			if canSpawn(cars, def) {
				var spawnCol int
				if def.Dir == DirRight {
					spawnCol = -CarWidth // enters from the left
				} else {
					spawnCol = GridCols // enters from the right
				}
				cars = append(cars, Car{Col: spawnCol})
			}

			// Deep-copy for the event so the lane can freely modify cars next tick.
			snapshot := make([]Car, len(cars))
			copy(snapshot, cars)

			select {
			case carEventsCh <- CarEvent{Lane: def.Row, Cars: snapshot}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// canSpawn returns true when the lane is clear enough near the spawn edge for a
// new car to appear without overlapping the most recently spawned vehicle.
func canSpawn(cars []Car, def LaneDef) bool {
	if len(cars) == 0 {
		return true
	}
	if def.Dir == DirRight {
		// Most recently spawned car has the smallest col value.
		minCol := cars[0].Col
		for _, c := range cars[1:] {
			if c.Col < minCol {
				minCol = c.Col
			}
		}
		return minCol >= def.SpawnGap
	}
	// Left-moving: most recently spawned car has the largest col value.
	maxCol := cars[0].Col
	for _, c := range cars[1:] {
		if c.Col > maxCol {
			maxCol = c.Col
		}
	}
	return maxCol <= GridCols-def.SpawnGap
}
