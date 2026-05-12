package main

import "time"

// state.go define as constantes do grid, tipos e estruturas usadas pelo jogo.
// Contém as definições de comandos, direção, eventos de carros e o GameState.

const (
	GridCols   = 20
	GridRows   = 10
	CarWidth   = 2
	StartRow   = 9
	StartCol   = 10
	GoalRow    = 0
	StartLives = 3
)

// LaneDef descreve como uma faixa deve se comportar.
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

// Command representa ações do jogador (teclado).
type Command int

const (
	CmdUp Command = iota
	CmdDown
	CmdLeft
	CmdRight
	CmdQuit
)

// Direction indica sentido dos carros na faixa.
type Direction int

const (
	DirRight Direction = iota
	DirLeft
)

// Car é um carro na faixa; somente a coluna é relevante aqui.
type Car struct {
	Col int
}

// CarEvent é o evento enviado por uma lane para o game loop com um snapshot
// dos carros naquela faixa. Enviamos uma cópia (slice copiado) para evitar
// compartilhamento de backing arrays entre goroutines.
type CarEvent struct {
	Lane int
	Cars []Car
}

// Status do jogo
type Status int

const (
	StatusPlaying Status = iota
	StatusWon
	StatusLost
)

// GameState é o estado completo do jogo mantido pelo game loop. Note que
// Cars é um array de slices (uma slice por linha). O game loop é o dono
// desses slices e envia cópias ao renderer para evitar data races.
type GameState struct {
	FrogRow int
	FrogCol int
	Lives   int
	Cars    [GridRows][]Car
	Status  Status
	Tick    uint64
}
