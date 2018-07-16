package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/klokare/evo"
	"github.com/klokare/evo/config"
	"github.com/klokare/evo/config/source"
	"github.com/klokare/evo/example"
	"github.com/klokare/evo/neat"
	"github.com/kpacha/neatflappy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// Parse the command-line flags
	var (
		iter  = flag.Int("iterations", 100, "number of iterations for experiment")
		cpath = flag.String("config", "neatflappy.json", "path to the configuration file")
		lpath = flag.String("training", "log.txt", "path to the training data file")
	)
	flag.Parse()

	src, err := source.NewJSONFromFile(*cpath)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	cfg := config.Configurer{Source: source.Multi([]config.Source{
		source.Flag{},        // Check flags  first
		source.Environment{}, // Then check environment variables
		src,                  // Lastly, consult the configuration file
	})}
	exp := neat.NewExperiment(cfg)
	exp.AddSubscription(evo.Subscription{Event: evo.Completed, Callback: example.ShowBest})
	// Run the experiment for a set number of iterations
	ctx, fn, cb := evo.WithIterations(context.Background(), *iter)
	defer fn() // ensure the context cancels
	exp.AddSubscription(evo.Subscription{Event: evo.Evaluated, Callback: cb})

	// Stop the experiment if there is a solution
	ctx, fn, cb = evo.WithSolution(ctx)
	defer fn() // ensure the context cancels
	exp.AddSubscription(evo.Subscription{Event: evo.Evaluated, Callback: cb})

	logData, err := ioutil.ReadFile(*lpath)
	if err != nil {
		log.Fatal("reading the training data:", err.Error())
		return
	}

	evaluator := neatflappy.TrainEvaluator{
		Log: logData,
	}

	// Execute the experiment
	if _, err = evo.Run(ctx, exp, evaluator); err != nil {
		log.Fatalf("%+v\n", err)
	}
}
