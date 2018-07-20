package neatflappy

import (
	"sync"

	"github.com/klokare/evo"
)

// Searcher evaluates phenomes all at once
type Searcher struct{}

// Search the solution space with the phenomes
func (s Searcher) Search(eval evo.Evaluator, phenomes []evo.Phenome) (results []evo.Result, err error) {
	// Receive results
	results = make([]evo.Result, 0, len(phenomes))
	ch := make(chan evo.Result, len(phenomes))
	ec := make(chan error, len(phenomes))
	wg := new(sync.WaitGroup)

	// Perform the tasks
	for _, p := range phenomes {
		wg.Add(1)
		go func(phenome evo.Phenome) {
			defer wg.Done()
			r, err := eval.Evaluate(phenome)
			if err != nil {
				ec <- err
				return
			}
			ch <- r
		}(p)
	}

	wg.Wait()
	close(ch)
	close(ec)

	for r := range ch {
		results = append(results, r)
	}

	for e := range ec {
		if e != nil {
			err = e
			break
		}
	}

	return
}
