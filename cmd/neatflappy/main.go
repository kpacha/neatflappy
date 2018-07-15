package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/klokare/evo"
	"github.com/klokare/evo/config"
	"github.com/klokare/evo/config/source"
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
	)
	flag.Parse()

	g := neatflappy.NewGame()

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
	exp.AddSubscription(evo.Subscription{Event: evo.Completed, Callback: neatflappy.ShowBest})
	// Run the experiment for a set number of iterations
	ctx, fn, cb := evo.WithIterations(context.Background(), *iter)
	defer fn() // ensure the context cancels
	exp.AddSubscription(evo.Subscription{Event: evo.Evaluated, Callback: cb})

	// Stop the experiment if there is a solution
	ctx, fn, cb = evo.WithSolution(ctx)
	defer fn() // ensure the context cancels
	exp.AddSubscription(evo.Subscription{Event: evo.Evaluated, Callback: cb})

	evaluator := neatflappy.Evaluator{
		Jumper:  g.Jumper,
		Fitness: g.Fitness,
	}

	go func() {
		// Execute the experiment
		if _, err = evo.Run(ctx, exp, evaluator); err != nil {
			log.Fatalf("%+v\n", err)
		}
	}()

	// On browsers, let's use fullscreen so that this is playable on any browsers.
	// It is planned to ignore the given 'scale' apply fullscreen automatically on browsers (#571).
	if runtime.GOARCH == "js" {
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.Run(g.Update(ctx), neatflappy.ScreenWidth, neatflappy.ScreenHeight, 1, "Flappy Gopher (Ebiten Demo)"); err != nil {
		panic(err)
	}
}
