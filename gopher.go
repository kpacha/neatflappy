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

func NewGopher() *Gopher {
	return &Gopher{fitness: make(chan float64)}
}

type Gopher struct {
	Name string
	// The gopher's position
	x16  int
	y16  int
	vy16 int

	successes int
	jumps     int

	jumper  Jumper
	fitness chan float64

	isDead bool
}

func (g *Gopher) init() {
	g.x16 = 0
	g.y16 = 100 * 16
	if g.jumper == nil {
		g.jumper = new(InteractiveJumper)
	}
	g.isDead = false
	g.jumps = 0
}

func (g *Gopher) score() float64 {
	distance := float64(g.x16) / 1600
	extra := float64(2+g.successes) / float64(g.jumps+1)
	return (distance*distance + extra*extra*extra) / 2
}

func (g *Gopher) jump(in []int) bool {
	offset := len(in)
	input := make([]float64, offset+3)
	for i := 0; i < offset/2; i++ {
		input[2*i] = float64(in[2*i]-4) / 8
		input[2*i+1] = float64(in[2*i+1]-4) / 8
	}
	input[offset] = (float64(g.y16)/16 + 300) / 600
	input[offset+1] = (float64(g.vy16) + 96) / 192
	input[offset+2] = 1

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
