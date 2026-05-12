package main

import "time"

const (
	GridCols   = 20
	GridRows   = 10
	CarWidth   = 2
	StartRow   = 9
	StartCol   = 10
	GoalRow    = 0
	StartLives = 3
)

type LaneDef struct {
	Row      int
	Dir      Direction
	Interval time.Duration
	SpawnGap int
}

var laneDefs = []LaneDef{
	{Row: 2, Dir: DirRight, Interval: 350 * time.Millisecond, SpawnGap: 6},
	{Row: 5, Dir: DirLeft, Interval: 250 * time.Millisecond, SpawnGap: 5},
	{Row: 7, Dir: DirRight, Interval: 450 * time.Millisecond, SpawnGap: 7},
}

type Command int

const (
	CmdUp Command = iota
	CmdDown
	CmdLeft
	CmdRight
	CmdQuit
)

type Direction int

const (
	DirRight Direction = iota
	DirLeft
)

type Car struct {
	Col int
}

type CarEvent struct {
	Lane int
	Cars []Car
}

type Status int

const (
	StatusPlaying Status = iota
	StatusWon
	StatusLost
)

type GameState struct {
	FrogRow int
	FrogCol int
	Lives   int
	Cars    [GridRows][]Car
	Status  Status
	Tick    uint64
}
