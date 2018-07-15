package bolt

import (
	"log"

	"github.com/klokare/evo"
)

type Evo struct {
	Client *Client
}

func (e *Evo) StoreBest(pop evo.Population) error {
	// Copy the genomes so we can sort them without affecting other listeners
	genomes := make([]evo.Genome, len(pop.Genomes))
	copy(genomes, pop.Genomes)

	// Sort so the best genome is at the end
	evo.SortBy(genomes, evo.BySolved, evo.ByFitness, evo.ByComplexity, evo.ByAge)

	// Output the best
	best := genomes[len(genomes)-1]

	log.Printf("storing: %s", best.Decoded.String())

	return e.Client.Update(PhenomeBucket, itob(uint64(best.ID)), best)
}
