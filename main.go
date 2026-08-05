package main

import (
	"log"
	"net/http"
	"io"
	"errors"
	"bytes"
	"math"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"encoding/json"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Transform int
const (
	Last Transform = iota
	Initial
	Current
)

const CurrentPage = -1

type PageAxisTransform struct {
	Last float64
	Initial float64
	Current float64
}

func (t *PageAxisTransform) CalculateLast() float64 {
	return t.Last + t.Current - t.Initial
}

type PageTransform struct {
	X, Y PageAxisTransform
	Scale float64
}

type PaginationPageHeight struct {
	Current float32
	Visual float32
}

type Game struct{
	Images []*ebiten.Image
	PageTransform []PageTransform
	CurrentPage int
	VisualPage float64 
	PaginationPageHeight []PaginationPageHeight
}

func (g *Game) GetCurrentPageTransform(pageIndex int) *PageTransform {
	if (pageIndex == CurrentPage) {
		pageIndex = g.CurrentPage
	}
	return &g.PageTransform[pageIndex]
}

func (g *Game) GetPageTransformDiff(pageIndex int) (float64, float64) {
	lastX, lastY := g.GetPageTransform(Last, pageIndex)
	initialX, initialY := g.GetPageTransform(Initial, pageIndex)
	currentX, currentY := g.GetPageTransform(Current, pageIndex)
	return lastX + currentX - initialX, lastY + currentY - initialY 
}

func (g *Game) PageCalculateLast(pageIndex int) (float64, float64) {
	transform := g.GetCurrentPageTransform(pageIndex)
	return transform.X.CalculateLast(), transform.Y.CalculateLast()
}

func (g *Game) GetPageTransform(prop Transform, pageIndex int) (float64, float64) {
	transform := g.GetCurrentPageTransform(pageIndex)
	switch prop {
	case Last:
		return transform.X.Last, transform.Y.Last
	case Initial:
		return transform.X.Initial, transform.Y.Initial
	case Current:
		return transform.X.Current, transform.Y.Current
	}

	return 0, 0
}

func (g *Game) SetPageTransform(prop Transform, x, y float64) {
	transform := g.GetCurrentPageTransform(CurrentPage)
	switch prop {
		case Last: {
			transform.X.Last = x
			transform.Y.Last = y
		}
		case Initial: {
			transform.X.Initial = x
			transform.Y.Initial = y
		}	
		case Current: {
			transform.X.Current = x
			transform.Y.Current = y
		}	
	}
}

func (g *Game) GetPageScale(pageIndex int) float64 {
	transform := g.GetCurrentPageTransform(pageIndex)
	return transform.Scale
}
func (g *Game) SetPageScale(value float64) {
	transform := g.GetCurrentPageTransform(CurrentPage)
	transform.Scale = value
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
    for i, img := range g.Images {
        imgWidth := float64(img.Bounds().Dx())
		x, _ := g.GetPageTransform(Last, i)
		if float64(outsideWidth) > imgWidth && g.GetPageScale(i) == 1.0 && x == 0 {
            g.PageTransform[i].X.Last = (float64(outsideWidth) - imgWidth) / 2
        }
    }
    return outsideWidth, outsideHeight
}

func (g *Game) PreviousPage() {
	if g.CurrentPage <= 0 {
		return
	}
	
	g.CurrentPage--
}
func (g *Game) NextPage() {
	if g.CurrentPage + 1 >= len(g.Images) {
		return
	}
	
	g.CurrentPage++
}
func (g *Game) UpdateAnimation() {
	g.VisualPage += (float64(g.CurrentPage) - g.VisualPage) * 0.1
}

func (g *Game) Update() error {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY) 
	scrollX, scrollY := ebiten.Wheel()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.SetPageTransform(Initial, mouseX, mouseY)
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.SetPageTransform(Current, mouseX, mouseY)
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		lastX, lastY := g.PageCalculateLast(CurrentPage)
		g.SetPageTransform(Last, lastX, lastY) 
		g.SetPageTransform(Current, 0, 0)
		g.SetPageTransform(Initial, 0, 0)
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.NextPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.PreviousPage()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
		oldScale := g.GetPageScale(CurrentPage)
		newScale := math.Max(0, oldScale + (scrollY / 10))

		if (oldScale > 0 && newScale > 0) {
			scaleFactor := newScale / oldScale

			lastX, lastY := g.GetPageTransform(Last, CurrentPage)
			
			newLastX := mouseX - (mouseX - lastX) * scaleFactor
			newLastY := mouseY - (mouseY - lastY) * scaleFactor

			g.SetPageTransform(Last, newLastX, newLastY)
			g.SetPageScale(newScale)
		}
	} else {
		var multiplier float64 = 50
		lastX, lastY := g.GetPageTransform(Last, CurrentPage)
		g.SetPageTransform(Last, lastX + (scrollX * multiplier), lastY + (scrollY * multiplier))
	}

	g.UpdateAnimation()
	return nil
}

func (g *Game) DrawPagination(screen *ebiten.Image) {
	screenWidth, screenHeight := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())

	for i, _ := range g.Images {
		color := color.RGBA{ R: 0, G: 0, B: 255, A: 55 }
		if i <= g.CurrentPage {
			color.A = 255 
		}
		var gap float32 = g.PaginationPageHeight[i].Visual 
		totalItems := float32(len(g.Images))
		size := (float32(screenWidth) - (gap * totalItems)) / totalItems
		x := size * float32(i)
		var height float32 = 8 
		vector.DrawFilledRect(screen, x + gap * float32(i), float32(screenHeight) - height, size, height, color, true)
	}
}

func (g *Game) DrawPages(screen *ebiten.Image) {
	screenWidth, screenHeight := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	
	for i, cImage := range g.Images {
		op := &ebiten.DrawImageOptions{}
		translateX, translateY := g.GetPageTransformDiff(i) 
		var scale float64 = 1

		imageWidth := float64(cImage.Bounds().Dx())
		if screenWidth < imageWidth {
			scale = screenWidth / imageWidth 
		}
		
		scale *= g.GetPageScale(i)
		op.GeoM.Scale(scale, scale)
		
		currentPageXOffset := (float64(i) - g.VisualPage) * screenWidth
		op.GeoM.Translate(translateX + currentPageXOffset, translateY)

		clipRect := image.Rect(int(currentPageXOffset), 0, int(currentPageXOffset + screenWidth), int(screenHeight))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)

		clipped.DrawImage(cImage, op)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.DrawPages(screen)
	g.DrawPagination(screen)
}

func LoadImageFromUrl(url string) (*ebiten.Image, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Failed to fetch")
	}
	
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

type MangedexChapter struct {
	Hash string `json:"hash"`
	Data []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}
type MangedexChapterResult struct {
	Result string `json:"result"`
	BaseUrl string `json:"baseUrl"`
	Chapter MangedexChapter
}

func FetchManga() (error, MangedexChapterResult) {
	var result MangedexChapterResult

	url := "https://api.mangadex.org/at-home/server/598ad0b5-110f-4946-a328-0e1755cfa189"
	res, err := http.Get(url)
	if err != nil {
		return err, result
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("Failed to fetch"), result
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return err, result 
	}
	return nil, result 
 }

func main() {
	var game Game
	err, chaptersResult := FetchManga()
	if err != nil {
		 return
	 }

	game.PageTransform = make([]PageTransform, len(chaptersResult.Chapter.Data))
	game.PaginationPageHeight = make([]PaginationPageHeight, len(chaptersResult.Chapter.Data))
	for i := range game.PageTransform {
    	game.PageTransform[i].Scale = 1.0
    	game.PaginationPageHeight[i] = PaginationPageHeight{
			Current: 4,
			Visual: 4,
		}
	}

	for _, chapterData := range chaptersResult.Chapter.Data {
		img, _ := LoadImageFromUrl(chaptersResult.BaseUrl + "/data/" + chaptersResult.Chapter.Hash + "/" + chapterData)
		game.Images = append(game.Images, img)
	}
	// img, _ := LoadImageFromUrl("https://uploads.mangadex.org/data/3303dd03ac8d27452cce3f2a882e94b2/2-2a5e95dfec7f15cd01f9a63835be18a22fb77a10fd2d62858c7dcbb6e6c622f9.png")
	// game.Images = append(game.Images, img)
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Hello, World!")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}

