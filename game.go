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
	"math/rand"

	"github.com/golang/freetype/truetype"
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
	ScreenWidth      = 640
	ScreenHeight     = 480
	tileSize         = 32
	fontSize         = 32
	smallFontSize    = fontSize / 2
	pipeWidth        = tileSize * 2
	pipeStartOffsetX = 8
	pipeIntervalX    = 8
	pipeGapY         = 5
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

	Gopher *Gopher

	// Camera
	cameraX int
	cameraY int

	// Pipes
	pipeTileYs []int

	Jumper  chan Jumper
	Fitness chan int

	iteration int

	speedFactor int
}

func NewGame(speedFactor int) *Game {
	g := &Game{
		Gopher:      &Gopher{},
		Jumper:      make(chan Jumper),
		Fitness:     make(chan int),
		speedFactor: speedFactor,
	}
	g.init()
	return g
}

func (g *Game) init() {
	g.Gopher.init()
	g.cameraX = -240
	g.cameraY = 0
	g.pipeTileYs = make([]int, 256)
	for i := range g.pipeTileYs {
		g.pipeTileYs[i] = rand.Intn(6) + 2
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

func (g *Game) Update(ctx context.Context) func(*ebiten.Image) error {
	return func(screen *ebiten.Image) error {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.speedFactor < 5000 {
			g.speedFactor += 10 * g.speedFactor / 100
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.speedFactor > 50 {
			g.speedFactor -= 10 * g.speedFactor / 100
		}
		switch g.mode {
		case ModeSetup:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case g.Gopher.jumper = <-g.Jumper:
				g.init()
				g.mode++
				g.iteration++
				g.Gopher.Name = fmt.Sprintf("gopher-%d", g.iteration)
			}

		case ModeGame:
			if g.update() {
				// hitPlayer.Rewind()
				// hitPlayer.Play()
				f := g.Gopher.score()
				g.Fitness <- f
				if f > 1000 {
					g.mode = ModeGameOver
				} else {
					g.mode = ModeSetup
				}
			}
		case ModeGameOver:
		}

		if ebiten.IsDrawingSkipped() {
			return nil
		}

		screen.Fill(color.RGBA{0x80, 0xa0, 0xc0, 0xff})
		g.drawTiles(screen)

		g.drawGopher(screen)

		var texts []string
		switch g.mode {
		case ModeGameOver:
			texts = []string{"", "GAMEOVER!"}
		}
		for i, l := range texts {
			x := (ScreenWidth - len(l)*fontSize) / 2
			text.Draw(screen, l, arcadeFont, x, (i+4)*fontSize, color.White)
		}

		scoreStr := fmt.Sprintf("%04d", g.Gopher.score())
		text.Draw(screen, scoreStr, arcadeFont, ScreenWidth-len(scoreStr)*fontSize, fontSize, color.White)
		ebitenutil.DebugPrint(
			screen,
			fmt.Sprintf("Speed: %d%. FPS: %0.2f. Gopher: %s", g.speedFactor, ebiten.CurrentFPS(), g.Gopher.Name),
		)
		return nil
	}
}

func (g *Game) update() bool {
	shloudJump := g.Gopher.jump(g.scan())
	g.Gopher.x16 += 32 * g.speedFactor / 100
	g.cameraX += 2 * g.speedFactor / 100
	if shloudJump {
		g.Gopher.vy16 = -96
		// jumpPlayer.Rewind()
		// jumpPlayer.Play()
	}
	g.Gopher.y16 += (g.Gopher.vy16 + 2*g.speedFactor/100) * g.speedFactor / 100

	// Gravity
	g.Gopher.vy16 += 4 * g.speedFactor / 100
	if g.Gopher.vy16 > 96 {
		g.Gopher.vy16 = 96
	}

	return g.hit()
}

func (g *Game) pipeAt(tileX int) (tileY int, ok bool) {
	if (tileX - pipeStartOffsetX) <= 0 {
		return 0, false
	}
	if floorMod(tileX-pipeStartOffsetX, pipeIntervalX) != 0 {
		return 0, false
	}
	idx := floorDiv(tileX-pipeStartOffsetX, pipeIntervalX)
	return g.pipeTileYs[idx%len(g.pipeTileYs)], true
}

func (g *Game) scan() []int {
	w, h := gopherImage.Size()
	x0 := floorDiv(g.Gopher.x16, 16) + (w-gopherWidth)/2
	y0 := floorDiv(g.Gopher.y16, 16) + (h-gopherHeight)/2
	y1 := y0 + gopherHeight
	res := []int{}
	if y0 < -tileSize*4 {
		return res
	}
	if y1 >= ScreenHeight-tileSize {
		return res
	}
	xMin := floorDiv(x0-pipeWidth, tileSize)

	for x := xMin; x < xMin+14; x++ {
		y, _ := g.pipeAt(x)
		res = append(res, y)
	}
	return res
}

func (g *Game) hit() bool {
	if g.mode != ModeGame {
		return false
	}

	w, h := gopherImage.Size()
	x0 := floorDiv(g.Gopher.x16, 16) + (w-gopherWidth)/2
	y0 := floorDiv(g.Gopher.y16, 16) + (h-gopherHeight)/2
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
	op := &ebiten.DrawImageOptions{}
	w, h := gopherImage.Size()
	op.GeoM.Translate(-float64(w)/2.0, -float64(h)/2.0)
	op.GeoM.Rotate(float64(g.Gopher.vy16) / 96.0 * math.Pi / 6)
	op.GeoM.Translate(float64(w)/2.0, float64(h)/2.0)
	op.GeoM.Translate(float64(g.Gopher.x16/16.0)-float64(g.cameraX), float64(g.Gopher.y16/16.0)-float64(g.cameraY))
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(gopherImage, op)
}
