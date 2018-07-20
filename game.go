package neatflappy

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/golang/freetype/truetype"
	"github.com/klokare/evo"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis"
	"github.com/hajimehoshi/ebiten/audio/wav"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	raudio "github.com/hajimehoshi/ebiten/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	resources "github.com/hajimehoshi/ebiten/examples/resources/images/flappy"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"
)

func floorDiv(x, y int) int {
	d := x / y
	if d*y == x || x >= 0 {
		return d
	}
	return d - 1
}

func floorMod(x, y int) int {
	return x - floorDiv(x, y)*y
}

const (
	ScreenWidth       = 640
	ScreenHeight      = 480
	tileSize          = 32
	fontSize          = 32
	smallFontSize     = fontSize / 2
	pipeWidth         = tileSize * 2
	pipeStartOffsetX  = 8
	pipeIntervalX     = 8
	pipeGapY          = 5
	solutionThreshold = 10000
)

var (
	gopherImage     *ebiten.Image
	tilesImage      *ebiten.Image
	arcadeFont      font.Face
	smallArcadeFont font.Face
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(resources.Gopher_png))
	if err != nil {
		log.Fatal(err)
	}
	gopherImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	img, _, err = image.Decode(bytes.NewReader(resources.Tiles_png))
	if err != nil {
		log.Fatal(err)
	}
	tilesImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

func init() {
	tt, err := truetype.Parse(fonts.ArcadeN_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	arcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	smallArcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    smallFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

var (
	audioContext *audio.Context
	jumpPlayer   *audio.Player
	hitPlayer    *audio.Player
)

func init() {
	audioContext, _ = audio.NewContext(44100)

	jumpD, err := vorbis.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Jump_ogg))
	if err != nil {
		log.Fatal(err)
	}
	jumpPlayer, err = audio.NewPlayer(audioContext, jumpD)
	if err != nil {
		log.Fatal(err)
	}

	jabD, err := wav.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Jab_wav))
	if err != nil {
		log.Fatal(err)
	}
	hitPlayer, err = audio.NewPlayer(audioContext, jabD)
	if err != nil {
		log.Fatal(err)
	}
}

type Mode int

const (
	ModeSetup Mode = iota
	ModeGame
	ModeGameOver
)

type Game struct {
	mode Mode

	Gopher []*Gopher

	// Camera
	cameraX int
	cameraY int

	Task chan Task

	NextPopulation chan evo.Population
	Population     *evo.Population

	level Level

	iteration      int
	maxRuns        int
	populationSize int

	speedFactor int
}

func NewGame(speedFactor, runs, populationSize int) *Game {
	g := &Game{
		Gopher:         make([]*Gopher, populationSize),
		Task:           make(chan Task, populationSize),
		NextPopulation: make(chan evo.Population, 1),
		speedFactor:    speedFactor,
		level:          Level1(0),
		maxRuns:        runs,
		populationSize: populationSize,
	}
	g.init()
	log.Printf("Game created with %d max runs and a population size of %d", runs, populationSize)
	return g
}

func (g *Game) init() {
	g.cameraX = -240
	g.cameraY = 0

	l := (g.iteration / g.populationSize) + 1
	if l < level2 {
		g.level = Level1(l)
		// } else if l < level3 {
		// 	g.level = Level2(l)
		// } else if l < level4 {
		// 	g.level = Level3(l)
		// } else if l < level5 {
		// 	if g.iteration%g.populationSize == 0 {
		// 		initPipeTileYs()
		// 	}
		// 	g.level = Level4(l)
		// } else if l < level6 {
		// 	if g.iteration%g.populationSize == 0 {
		// 		initPipeTileYs()
		// 	}
		// 	g.level = Level5(l)
	} else {
		if (g.iteration/g.populationSize)%20 == 0 {
			initPipeTileYs()
		}
		g.level = Level6(l)
	}
}

func jump() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if len(inpututil.JustPressedTouchIDs()) > 0 {
		return true
	}
	return false
}

func (g *Game) initGopher(task Task) {
	if g.Gopher[g.iteration%g.populationSize] == nil {
		g.Gopher[g.iteration%g.populationSize] = NewGopher()
	}
	g.Gopher[g.iteration%g.populationSize].init()
	g.Gopher[g.iteration%g.populationSize].jumper = task.Jumper
	g.Gopher[g.iteration%g.populationSize].fitness = task.Fitness
	g.Gopher[g.iteration%g.populationSize].Name = fmt.Sprintf("gopher-%d", g.iteration)
}

func (g *Game) ModeSetup(ctx context.Context, screen *ebiten.Image) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case pop := <-g.NextPopulation:
		g.Population = &pop
	case task := <-g.Task:
		g.initGopher(task)
		g.iteration++
		if g.iteration%g.populationSize == 0 {
			g.mode = ModeGame
			return nil
		}
	default:
	}
	return nil
}

func (g *Game) ModeGameOver(ctx context.Context, screen *ebiten.Image) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case pop := <-g.NextPopulation:
		g.Population = &pop
	default:
	}
	return nil
}

func (g *Game) changeModeToSetup() {
	g.mode = ModeSetup
	g.init()
}

func (g *Game) Update(ctx context.Context) func(*ebiten.Image) error {
	return func(screen *ebiten.Image) error {
		g.checkSpeed()

		score := 0
		switch g.mode {
		case ModeSetup:
			if err := g.ModeSetup(ctx, screen); err != nil {
				return err
			}
		case ModeGameOver:
			log.Println("game over!")
			if err := g.ModeGameOver(ctx, screen); err != nil {
				return err
			}

		case ModeGame:
			totalDeads := 0
			bestFitness := 0
			g.cameraX += 2
			_, successed := g.pipeAt(g.cameraX - 2)
			successed = successed && (g.cameraX > pipeStartOffsetX) && (floorMod(g.cameraX-pipeStartOffsetX, pipeIntervalX) < 2)
			for _, gopher := range g.Gopher {
				if gopher.isDead {
					totalDeads++
					continue
				}
				g.update(gopher)
				dead := g.hit(gopher)
				// if dead {
				// hitPlayer.Rewind()
				// hitPlayer.Play()
				// }
				f := gopher.score()
				fInt := int(f)
				if fInt > g.level.ExitScore() || dead {
					gopher.fitness <- f
					gopher.isDead = true
				}
				if fInt > bestFitness {
					bestFitness = fInt
					score = fInt
				}
				if successed {
					gopher.successes++
				}
			}
			if totalDeads == g.populationSize {
				if bestFitness > 100*solutionThreshold {
					g.mode = ModeGameOver
				} else {
					g.changeModeToSetup()
				}
			}
		}

		if ebiten.IsDrawingSkipped() {
			return nil
		}

		screen.Fill(color.RGBA{0x80, 0xa0, 0xc0, 0xff})
		g.drawTiles(screen)

		if g.mode != ModeSetup {
			g.drawGopher(screen)
		}

		if g.Population != nil {
			g.drawPopulation(screen)
		}

		var texts []string
		switch g.mode {
		case ModeSetup:
			generation := fmt.Sprintf("GENERATION #%d", 1+g.iteration/g.populationSize)
			status := fmt.Sprintf("%d/%d", g.iteration%g.populationSize, g.populationSize)
			texts = []string{"BUILDING", generation, "", status, "", "WAIT FOR IT..."}
		case ModeGameOver:
			texts = []string{"", "GAMEOVER!"}
		}
		for i, l := range texts {
			x := (ScreenWidth - len(l)*fontSize) / 2
			text.Draw(screen, l, arcadeFont, x, (i+4)*fontSize, color.White)
		}

		scoreStr := fmt.Sprintf("%04d", score)
		text.Draw(screen, scoreStr, arcadeFont, ScreenWidth-len(scoreStr)*fontSize, fontSize, color.White)
		ebitenutil.DebugPrint(
			screen,
			fmt.Sprintf(
				"Speed: %d%%. FPS: %0.2f. %s [%d]",
				g.speedFactor, ebiten.CurrentFPS(), g.level.String(), g.populationSize),
		)
		return nil
	}
}

func (g *Game) checkSpeed() {
	for k, v := range speedKeys {
		if inpututil.IsKeyJustPressed(k) {
			g.speedFactor = v
			ebiten.SetMaxTPS(60 * g.speedFactor / 100)
			return
		}
	}
}

func (g *Game) update(gopher *Gopher) {
	shloudJump := gopher.jump(g.scan(gopher))
	gopher.x16 += 32
	if shloudJump {
		gopher.jumps++
		gopher.vy16 = -96
		// jumpPlayer.Rewind()
		// jumpPlayer.Play()
	}
	gopher.y16 += gopher.vy16 + 2

	// Gravity
	gopher.vy16 += 4
	if gopher.vy16 > 96 {
		gopher.vy16 = 96
	}
}

func (g *Game) pipeAt(tileX int) (tileY int, ok bool) {
	return g.level.PipeAt(tileX)
}

func (g *Game) scan(gopher *Gopher) []int {
	w, h := gopherImage.Size()
	x0 := floorDiv(gopher.x16, 16) + (w-gopherWidth)/2
	y0 := floorDiv(gopher.y16, 16) + (h-gopherHeight)/2
	y1 := y0 + gopherHeight
	res := []int{8, 0, 8, 0}
	if y0 < -tileSize*4 {
		return res
	}
	if y1 >= ScreenHeight-tileSize {
		return res
	}
	xMin := floorDiv(x0-pipeWidth, tileSize)

	for x := xMin; x < xMin+14; x++ {
		if y, ok := g.pipeAt(x); ok {
			res = append(res, x-xMin-7, y)
		}
	}
	if len(res) == 4 {
		return res
	}
	return res[len(res)-4:]
}

func (g *Game) hit(gopher *Gopher) bool {
	if g.mode != ModeGame {
		return false
	}

	w, h := gopherImage.Size()
	x0 := floorDiv(gopher.x16, 16) + (w-gopherWidth)/2
	y0 := floorDiv(gopher.y16, 16) + (h-gopherHeight)/2
	x1 := x0 + gopherWidth
	y1 := y0 + gopherHeight
	if y0 < -tileSize*4 {
		return true
	}
	if y1 >= ScreenHeight-tileSize {
		return true
	}
	xMin := floorDiv(x0-pipeWidth, tileSize)
	xMax := floorDiv(x0+gopherWidth, tileSize)

	for x := xMin; x <= xMax; x++ {
		y, ok := g.pipeAt(x)
		if !ok {
			continue
		}
		if x0 >= x*tileSize+pipeWidth {
			continue
		}
		if x1 < x*tileSize {
			continue
		}
		if y0 < y*tileSize {
			return true
		}
		if y1 >= (y+pipeGapY)*tileSize {
			return true
		}
	}
	return false
}

func (g *Game) drawTiles(screen *ebiten.Image) {
	const (
		nx           = ScreenWidth / tileSize
		ny           = ScreenHeight / tileSize
		pipeTileSrcX = 128
		pipeTileSrcY = 192
	)

	op := &ebiten.DrawImageOptions{}
	for i := -2; i < nx+1; i++ {
		// ground
		op.GeoM.Reset()
		op.GeoM.Translate(float64(i*tileSize-floorMod(g.cameraX, tileSize)),
			float64((ny-1)*tileSize-floorMod(g.cameraY, tileSize)))
		r := image.Rect(0, 0, tileSize, tileSize)
		op.SourceRect = &r
		screen.DrawImage(tilesImage, op)

		// pipe
		if tileY, ok := g.pipeAt(floorDiv(g.cameraX, tileSize) + i); ok {
			for j := 0; j < tileY; j++ {
				op.GeoM.Reset()
				op.GeoM.Scale(1, -1)
				op.GeoM.Translate(float64(i*tileSize-floorMod(g.cameraX, tileSize)),
					float64(j*tileSize-floorMod(g.cameraY, tileSize)))
				op.GeoM.Translate(0, tileSize)
				if j == tileY-1 {
					r := image.Rect(pipeTileSrcX, pipeTileSrcY, pipeTileSrcX+tileSize*2, pipeTileSrcY+tileSize)
					op.SourceRect = &r
				} else {
					r := image.Rect(pipeTileSrcX, pipeTileSrcY+tileSize, pipeTileSrcX+tileSize*2, pipeTileSrcY+tileSize*2)
					op.SourceRect = &r
				}
				screen.DrawImage(tilesImage, op)
			}
			for j := tileY + pipeGapY; j < ScreenHeight/tileSize-1; j++ {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(i*tileSize-floorMod(g.cameraX, tileSize)),
					float64(j*tileSize-floorMod(g.cameraY, tileSize)))
				if j == tileY+pipeGapY {
					r := image.Rect(pipeTileSrcX, pipeTileSrcY, pipeTileSrcX+pipeWidth, pipeTileSrcY+tileSize)
					op.SourceRect = &r
				} else {
					r := image.Rect(pipeTileSrcX, pipeTileSrcY+tileSize, pipeTileSrcX+pipeWidth, pipeTileSrcY+tileSize+tileSize)
					op.SourceRect = &r
				}
				screen.DrawImage(tilesImage, op)
			}
		}
	}
}

func (g *Game) drawGopher(screen *ebiten.Image) {
	for _, gopher := range g.Gopher {
		if gopher == nil || gopher.x16/16 < g.cameraX-3 {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		w, h := gopherImage.Size()
		op.GeoM.Translate(-float64(w)/2.0, -float64(h)/2.0)
		op.GeoM.Rotate(float64(gopher.vy16) / 96.0 * math.Pi / 6)
		op.GeoM.Translate(float64(w)/2.0, float64(h)/2.0)
		op.GeoM.Translate(float64(gopher.x16/16.0)-float64(g.cameraX), float64(gopher.y16/16.0)-float64(g.cameraY))
		if gopher.isDead {
			op.ColorM.Translate(100, 0, 0, 0)
		}
		op.Filter = ebiten.FilterLinear
		screen.DrawImage(gopherImage, op)
	}
}

func (g *Game) drawPopulation(screen *ebiten.Image) {
	// op := &ebiten.DrawImageOptions{}
	// w, h := gopherImage.Size()
	// op.GeoM.Translate(-float64(w)/2.0, -float64(h)/2.0)
	// op.GeoM.Rotate(float64(g.Gopher.vy16) / 96.0 * math.Pi / 6)
	// op.GeoM.Translate(float64(w)/2.0, float64(h)/2.0)
	// op.GeoM.Translate(float64(g.Gopher.x16/16.0)-float64(g.cameraX), float64(g.Gopher.y16/16.0)-float64(g.cameraY))
	// op.Filter = ebiten.FilterLinear
	// screen.DrawImage(gopherImage, op)

	// for _, genome := range g.Population.Genomes {
	// 	log.Printf("genome %d: %s", genome.ID, genome.Encoded.String())
	// }
}

var speedKeys = map[ebiten.Key]int{
	ebiten.KeyF1:  100,
	ebiten.KeyF2:  200,
	ebiten.KeyF3:  300,
	ebiten.KeyF4:  400,
	ebiten.KeyF5:  500,
	ebiten.KeyF6:  600,
	ebiten.KeyF7:  700,
	ebiten.KeyF8:  800,
	ebiten.KeyF9:  900,
	ebiten.KeyF10: 1000,
}
