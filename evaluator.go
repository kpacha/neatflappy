package neatflappy

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/klokare/evo"
	"gonum.org/v1/gonum/mat"
)

// Evaluator runs the flappy experiment
type Evaluator struct {
	Task       chan Task
	Population chan evo.Population
}

// Evaluate the flappy experiment with this phenome
func (e Evaluator) Evaluate(p evo.Phenome) (r evo.Result, err error) {
	t := Task{
		ID:      p.ID,
		Jumper:  &evoJumper{p},
		Fitness: make(chan float64),
	}
	e.Task <- t

	f := <-t.Fitness

	return evo.Result{
		ID:      p.ID,
		Fitness: f * f,
		Solved:  int(f) > 1000*solutionThreshold,
	}, nil
}

func (e Evaluator) PreSearch(pop evo.Population) error {
	go func() { e.Population <- pop }()
	return nil
}

func (e Evaluator) PostSearch(pop evo.Population) error {
	return nil
}

type Task struct {
	ID      int64
	Jumper  Jumper
	Fitness chan float64
}

type evoJumper struct {
	p evo.Phenome
}

func (e *evoJumper) Jump(input []float64) bool {
	in := mat.NewDense(1, len(input), input)
	outputs, err := e.p.Activate(in)
	if err != nil {
		return false
	}

	var out []float64
	if rcv, ok := outputs.(mat.RawColViewer); ok {
		out = rcv.RawColView(0)
	} else {
		out = make([]float64, 1)
		out[0] = outputs.At(0, 0)
	}

	return out[0] > 0.5
}

type TrainEvaluator struct {
	Log []byte
}

// Evaluate the flappy experiment with this phenome
func (e TrainEvaluator) Evaluate(p evo.Phenome) (r evo.Result, err error) {
	log.Println("training phenome", p.ID)
	samples := loadTrainingData(e.Log)

	if len(samples) == 0 {
		log.Fatal("no training data!")
	}

	input := []float64{}
	for _, sample := range samples {
		for _, d := range sample.In {
			input = append(input, float64(d))
		}
	}

	in := mat.NewDense(len(samples), len(samples[0].In), input)
	outputs, err := p.Activate(in)
	if err != nil {
		log.Fatal("error training:", err.Error())
	}

	out := make([]float64, len(samples))
	if rcv, ok := outputs.(mat.RawColViewer); ok {
		for k := range out {
			out[k] = rcv.RawColView(k)[0]
		}
	} else {
		for k := range out {
			out[k] = outputs.At(k, 0)
		}
	}

	oks := len(samples)
	for i, sample := range samples {
		if sample.Out != (out[i] > .5) {
			oks--
		}
	}

	solved := oks > 9999*len(samples)/10000
	log.Printf("phenome: %06d, oks: [%d/%d] solved: %v", p.ID, oks, len(samples), solved)

	return evo.Result{
		ID:      p.ID,
		Fitness: float64(oks * oks),
		Solved:  solved,
	}, nil
}

func loadTrainingData(in []byte) []Trace {
	decoder := json.NewDecoder(bytes.NewBuffer(in))
	samples := []Trace{}
	for {
		data := Trace{}
		if err := decoder.Decode(&data); err != nil {
			break
		}
		samples = append(samples, data)
	}
	return samples
}
