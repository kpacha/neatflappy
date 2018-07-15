package neatflappy

import (
	"encoding/json"
	"io"
	"log"
)

const (
	gopherWidth  = 30
	gopherHeight = 60
)

type Gopher struct {
	Name string
	// The gopher's position
	x16  int
	y16  int
	vy16 int

	jumper Jumper
}

func (g *Gopher) init() {
	g.x16 = 0
	g.y16 = 100 * 16
	if g.jumper == nil {
		g.jumper = new(InteractiveJumper)
	}
}

func (g *Gopher) score() int {
	x := floorDiv(g.x16, 16) / tileSize
	if (x - pipeStartOffsetX) <= 0 {
		return 0
	}
	return 10*floorDiv(x-pipeStartOffsetX, pipeIntervalX) + g.x16/1600
}

func (g *Gopher) jump(in []int) bool {
	offset := len(in)
	input := make([]float64, offset*6+2)
	for i := range input {
		input[i] = -1
	}
	for i, k := range in {
		if k == 0 {
			continue
		}
		input[i*6+(k-2)] = 1
	}
	input[offset*6] = (float64(g.y16) / 16) / 300
	input[offset*6+1] = float64(g.vy16) / 96

	return g.jumper.Jump(input)
}

type Jumper interface {
	Jump([]float64) bool
}

type InteractiveJumper int

func (InteractiveJumper) Jump(_ []float64) bool {
	return jump()
}

type InteractiveLogJumper struct {
	Out io.Writer
}

func (i InteractiveLogJumper) Jump(in []float64) bool {
	out := jump()
	data := Trace{
		In:  in,
		Out: out,
	}
	if err := json.NewEncoder(i.Out).Encode(data); err != nil {
		log.Println("error logging the game:", err.Error())
	}
	return out
}

type Trace struct {
	In  []float64
	Out bool
}
