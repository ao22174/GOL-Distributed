package stubs

import "uk.ac.bris.cs/gameoflife/util"

var GameOfLifeUpdate = "GameOfLifeOperations.Update"
var Pause = "GameOfLifeOperations.Pause"
var GameOfLifeAlive = "GameOfLifeOperations.AliveCells"
var Quit = "GameOfLifeOperations.Quit"
var SuperQuit = "GameOfLifeOperations.SuperQuit"

type Request struct {
	World       [][]byte
	Turns       int
	ImageHeight int
	ImageWidth  int
	Threads     int
}

type Response struct {
	Alive          []util.Cell
	AliveCount     int
	TurnsCompleted int
	FinalWorld     [][]uint8
	CurrentWorld   [][]uint8
}
