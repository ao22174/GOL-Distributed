package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var currentWorld [][]byte
var turnsdonec int
var waiting = make(chan bool)
var quitting = make(chan bool)
var superquit = make(chan bool)

//the bread and butter of the worker, calculates the next state of the Game Of Life
func calculateNextState(p gol.Params, world [][]byte) [][]byte {
	// create a new state
	newWorld := make([][]byte, p.ImageWidth)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageHeight)
	}

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[x][y] == 0 {
				if calculateSurroundings(x, y, world) == 3 {
					newWorld[x][y] = 255
				}
			}
			if world[x][y] == 255 {
				if calculateSurroundings(x, y, world) < 2 || calculateSurroundings(x, y, world) > 3 {
					newWorld[x][y] = 0
				}
				if calculateSurroundings(x, y, world) == 2 || calculateSurroundings(x, y, world) == 3 {
					newWorld[x][y] = world[x][y]
				}
			}
		}
	}
	return newWorld
}

func calculateSurroundings(row, column int, world [][]byte) int {
	count := 0
	rowAbove := row - 1
	rowBelow := row + 1
	if row == 0 {
		rowAbove = len(world[0]) - 1
	} else if row == len(world[0])-1 {
		rowBelow = 0
	}
	columnLeft := column - 1
	columnRight := column + 1
	if column == 0 {
		columnLeft = len(world[0]) - 1
	} else if column == len(world[0])-1 {
		columnRight = 0
	}
	surroundings := []byte{world[rowAbove][columnLeft], world[rowAbove][column], world[rowAbove][columnRight],
		world[row][columnLeft], world[row][columnRight], world[rowBelow][columnLeft], world[rowBelow][column],
		world[rowBelow][columnRight]}
	for _, surrounding := range surroundings {
		if surrounding == 255 {
			count = count + 1
		}
	}
	return count
}

// calculates the cell positions that are alive and passes it back as a list of cells
func calculateAliveCells(p gol.Params, world [][]byte) []util.Cell {
	var cs []util.Cell

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			if world[i][j] != 0 {
				cs = append(cs, util.Cell{X: j, Y: i})
			}
		}
	}

	return cs
}

// GameOfLifeOperations This is the method that is going to be RPC Called
type GameOfLifeOperations struct {
}

func (s *GameOfLifeOperations) Update(req stubs.Request, res *stubs.Response) (err error) {
	p := gol.Params{Turns: req.Turns, Threads: req.Threads, ImageWidth: req.ImageWidth, ImageHeight: req.ImageHeight}
	println(req.World)
	world := req.World
	currentWorld = make([][]byte, p.ImageWidth)
	for i := range world {
		currentWorld[i] = make([]byte, p.ImageHeight)
	}

	res.CurrentWorld = make([][]byte, p.ImageWidth)
	for i := range world {
		res.CurrentWorld[i] = make([]byte, p.ImageHeight)
	}
turnloop:
	for turn := 0; turn < p.Turns; turn++ {
		select {
		case <-quitting:
			break turnloop

		case <-waiting:
			fmt.Println("State paused")
			<-waiting
			fmt.Println("Loop resumed")
		default:
			world = calculateNextState(p, world)
			res.TurnsCompleted = turn
			turnsdonec = turn
			for i := 0; i < p.ImageHeight; i++ {
				for j := 0; j < p.ImageWidth; j++ {
					readWorld := world[i][j]
					res.CurrentWorld[i][j] = readWorld
					currentWorld[i][j] = readWorld
				}
			}
		}
	}

	res.FinalWorld = make([][]byte, p.ImageWidth)
	for i := range world {
		res.FinalWorld[i] = make([]byte, p.ImageHeight)
	}

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			readWorld := world[i][j]
			res.FinalWorld[i][j] = readWorld

		}
	}
	res.Alive = calculateAliveCells(p, world)
	return
}

func (s *GameOfLifeOperations) AliveCells(req stubs.Request, res *stubs.Response) (err error) {
	p := gol.Params{Turns: req.Turns, Threads: req.Threads, ImageWidth: req.ImageWidth, ImageHeight: req.ImageHeight}
	res.CurrentWorld = make([][]byte, p.ImageWidth)
	for i := range currentWorld {
		res.CurrentWorld[i] = make([]byte, p.ImageHeight)
	}
	println("reading alive cells")
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			readWorld := currentWorld[i][j]
			res.CurrentWorld[i][j] = readWorld
		}
	}
	println(res.CurrentWorld)
	if len(currentWorld) > 0 {
		res.AliveCount = len(calculateAliveCells(p, res.CurrentWorld))
		res.TurnsCompleted = turnsdonec
	} else {
		res.AliveCount = 0
	}
	return
}

func (s *GameOfLifeOperations) Pause(req stubs.Request, res *stubs.Response) (err error) {
	waiting <- true
	return
}

func (s *GameOfLifeOperations) Quit(req stubs.Request, res *stubs.Response) (err error) {
	println("quit")
	quitting <- true
	return
}

func (s *GameOfLifeOperations) SuperQuit(req stubs.Request, res *stubs.Response) (err error) {
	println("quit")
	quitting <- true
	superquit <- true
	return
}

//will be running separately on its own machine (AWS):
//the address will need to be found on the AWS machine, and flagged in the distributor
func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOfLifeOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	go func() {
	serverloop:
		for {
			print("loop")
			select {
			case <-superquit:
				println("Shutting down")
				listener.Close()
				break serverloop

			}
		}
	}()
	rpc.Accept(listener)
	return
}
