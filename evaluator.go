package neatflappy

import (
	"bytes"
	"encoding/json"
	"log"
	"math"

	"github.com/klokare/evo"
	"gonum.org/v1/gonum/mat"
)

// Evaluator runs the flappy experiment
type Evaluator struct {
	Jumper  chan Jumper
	Fitness chan int
}

// Evaluate the flappy experiment with this phenome
func (e Evaluator) Evaluate(p evo.Phenome) (r evo.Result, err error) {
	e.Jumper <- &evoJumper{p}

	f := <-e.Fitness

	return evo.Result{
		ID:      p.ID,
		Fitness: float64(f),
		Solved:  f > 100000,
	}, nil
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
		Fitness: math.Pow(float64(oks), 2),
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

// ShowBest is an EVO listener which will output a summary of the best genome in the population to the log
func ShowBest(pop evo.Population) error {

	// Copy the genomes so we can sort them without affecting other listeners
	genomes := make([]evo.Genome, len(pop.Genomes))
	copy(genomes, pop.Genomes)

	// Sort so the best genome is at the end
	evo.SortBy(genomes, evo.BySolved, evo.ByFitness, evo.ByComplexity, evo.ByAge)

	// Output the best
	best := genomes[len(genomes)-1]
	log.Printf("generation %d, id %d, species %d, fitness %f, solved %t, complexity %d\n", pop.Generation, best.ID, best.Species, best.Fitness, best.Solved, best.Complexity())
	return nil
}
