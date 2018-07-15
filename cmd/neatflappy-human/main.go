package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/kpacha/neatflappy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	g := neatflappy.NewGame(1)
	if runtime.GOARCH == "js" {
		ebiten.SetFullscreen(true)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file, err := os.Create("./log.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	go func() {
		g.Jumper <- neatflappy.InteractiveLogJumper{Out: file}
		f := <-g.Fitness
		log.Println("fitness:", f)
		time.Sleep(5 * time.Second)
		cancel()
	}()

	if err := ebiten.Run(g.Update(ctx), neatflappy.ScreenWidth, neatflappy.ScreenHeight, 1, "Flappy Gopher (Ebiten Demo)"); err != nil {
		panic(err)
	}
}
