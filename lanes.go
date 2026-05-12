package main

import (
	"context"
	"sync"
	"time"
)

func runLane(ctx context.Context, wg *sync.WaitGroup, def LaneDef, carEventsCh chan<- CarEvent) {
	defer wg.Done()
	ticker := time.NewTicker(def.Interval)
	defer ticker.Stop()
	var cars []Car
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var moved []Car
			for _, c := range cars {
				nc := c
				if def.Dir == DirRight {
					nc.Col++
				} else {
					nc.Col--
				}
				if nc.Col < GridCols && nc.Col+CarWidth > 0 {
					moved = append(moved, nc)
				}
			}
			cars = moved
			if canSpawn(cars, def) {
				var spawnCol int
				if def.Dir == DirRight {
					spawnCol = -CarWidth
				} else {
					spawnCol = GridCols
				}
				cars = append(cars, Car{Col: spawnCol})
			}
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

func canSpawn(cars []Car, def LaneDef) bool {
	if len(cars) == 0 {
		return true
	}
	if def.Dir == DirRight {
		minCol := cars[0].Col
		for _, c := range cars[1:] {
			if c.Col < minCol {
				minCol = c.Col
			}
		}
		return minCol >= def.SpawnGap
	}
	maxCol := cars[0].Col
	for _, c := range cars[1:] {
		if c.Col > maxCol {
			maxCol = c.Col
		}
	}
	return maxCol <= GridCols-def.SpawnGap
}


