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

type Screen int
const (
	MangaScreen Screen = iota
	ChapterScreen
)

type Transform int
const (
	Last Transform = iota
	Initial
	Current
)

const CurrentPage = -1

type PageAxisTransformLast struct {
 	Real float64
	Visual float64
}

type PageAxisTransform struct {
	Last PageAxisTransformLast 
	Initial float64
	Current float64
}

func (t *PageAxisTransform) CalculateLast() float64 {
	return t.Last.Real + t.Current - t.Initial
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

	ScreenHeight float64
	ScreenWidth float64

	CurrentScreen Screen
}

func (g *Game) GetCurrentPageTransform(pageIndex int) *PageTransform {
	if (pageIndex == CurrentPage) {
		pageIndex = g.CurrentPage
	}
	return &g.PageTransform[pageIndex]
}

func (g *Game) HasPageMouseMoved() bool {
	initialX, initialY := g.GetPageTransform(Initial, CurrentPage)
	currentX, currentY := g.GetPageTransform(Current, CurrentPage)
	return initialX != currentX || initialY != currentY
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
		return transform.X.Last.Visual, transform.Y.Last.Visual
	case Initial:
		return transform.X.Initial, transform.Y.Initial
	case Current:
		return transform.X.Current, transform.Y.Current
	}

	return 0, 0
}

func (g *Game) SetPageTransform(prop Transform, x, y float64, pageIndex int) {
	transform := g.GetCurrentPageTransform(pageIndex)
	switch prop {
		case Last: {
			transform.X.Last.Real = x
			transform.Y.Last.Real = y
			
			transform.X.Last.Visual = x
			transform.Y.Last.Visual = y
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

func (g *Game) CenterPages() {
	for i, img := range g.Images {
    	imgWidth, imgHeight := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		g.PageTransform[i].Scale = 1
		if imgWidth < g.ScreenWidth {
			g.SetPageTransform(Last, (g.ScreenWidth - imgWidth) / 2, 0, i)
		} else {
			scale := g.ScreenWidth / imgWidth 
			g.SetPageTransform(Last, 0, (g.ScreenHeight - imgHeight * scale) / 2, i)
		}
	}
}
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	hasResized := g.ScreenWidth != float64(outsideWidth) || g.ScreenHeight != float64(outsideHeight)

	g.ScreenWidth = float64(outsideWidth)
    g.ScreenHeight = float64(outsideHeight)

	if hasResized {
		g.CenterPages()
	}

	return outsideWidth, outsideHeight
}

func (g *Game) NavigateTo(target int) error {
		if target < 0 || target + 1 > len(g.Images) {
		return errors.New("Target out of bounds")
	}
	
	g.CurrentPage = target
	return nil
}

func (g *Game) PreviousPage() {
	g.NavigateTo(g.CurrentPage - 1)
}
func (g *Game) NextPage() {
	g.NavigateTo(g.CurrentPage + 1)
}
func (g *Game) UpdateChapterAnimation() {
	g.VisualPage += (float64(g.CurrentPage) - g.VisualPage) * 0.1
	for i, _ := range g.Images {
		pageHeight := g.PaginationPageHeight[i]
		g.PaginationPageHeight[i].Visual += (pageHeight.Current - pageHeight.Visual) * 0.12

		pageYLast := g.PageTransform[i].Y.Last
		g.PageTransform[i].Y.Last.Visual += (pageYLast.Real - pageYLast.Visual) * 0.14
	}
}

func (g *Game) ChapterPaginationUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY) 
	
	if mouseY < g.ScreenHeight - 80	{
		for i, _ := range g.Images {
			g.PaginationPageHeight[i].Current = 0  
		}
		
		return
	} 

	for i := 0; i < len(g.Images); i++ {
		var defaultHeight float32 = 10
		var aroundHoveredHeight float32 = defaultHeight + 10 
		var hoveredHeight float32 = aroundHoveredHeight + 12

		g.PaginationPageHeight[i].Current = defaultHeight 
		if mouseY > g.ScreenHeight - float64(hoveredHeight) {
			size, x := g.PageSize(i)

			if mouseX > float64(x) && mouseX < float64(x + size) {
				if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
					g.NavigateTo(i)
				}

				isFirst := i == 0
				if !isFirst {
					g.PaginationPageHeight[i - 1].Current = aroundHoveredHeight 
				}
				
				g.PaginationPageHeight[i].Current = hoveredHeight 

				isLast := len(g.Images) == i + 1 
				if !isLast {
					g.PaginationPageHeight[i + 1].Current = aroundHoveredHeight 
					i++
				}
			}
		} 
	}
		
}
func (g *Game) ChapterPageUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY) 
	_, scrollY := ebiten.Wheel()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.SetPageTransform(Initial, mouseX, mouseY, CurrentPage)
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.SetPageTransform(Current, mouseX, mouseY, CurrentPage)
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		hasPageMouseMoved := g.HasPageMouseMoved()
		if hasPageMouseMoved {
			lastX, lastY := g.PageCalculateLast(CurrentPage)
			g.SetPageTransform(Last, lastX, lastY, CurrentPage) 
			g.SetPageTransform(Current, 0, 0, CurrentPage)
			g.SetPageTransform(Initial, 0, 0, CurrentPage)
		} else {
			center := g.ScreenWidth / 2
			if mouseX < center {
				g.PreviousPage()
			} else {
				g.NextPage()
			}
		}
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

			g.SetPageTransform(Last, newLastX, newLastY, CurrentPage)
			g.SetPageScale(newScale)
		}
	} else {
		var multiplier float64 = 72
		var keyboardMultiplier float64 = 32
		// lastX, lastY := g.GetPageTransform(Last, CurrentPage)
		lastY := g.PageTransform[g.CurrentPage].Y.Last.Real
		// g.PageTransform[g.CurrentPage].X.Last.Real = lastX + (scrollX * multiplier)

		yScrollAmount := lastY + (scrollY * multiplier)
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			yScrollAmount -= keyboardMultiplier 
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			yScrollAmount += keyboardMultiplier 
		}
		g.PageTransform[g.CurrentPage].Y.Last.Real = yScrollAmount
	}


}
func (g *Game) Update() error {
	switch g.CurrentScreen {
		case ChapterScreen: {
			g.ChapterPaginationUpdate()
			g.ChapterPageUpdate()
			g.UpdateChapterAnimation()		
		}	
	}

	return nil
}

func (g *Game) PageSize(i int) (float32, float32) {
	var gap float32 = 6 
	totalItems := float32(len(g.Images))
	size := (float32(g.ScreenWidth) - (gap * totalItems)) / totalItems
	x := (size * float32(i)) + (gap * float32(i))
	return size, x
}

func (g *Game) DrawPagination(screen *ebiten.Image) {
	for i, _ := range g.Images {
		w, x := g.PageSize(i) 
		h := float32(g.PaginationPageHeight[i].Visual) 
		y := float32(g.ScreenHeight) - h 
		vector.FillRect(screen, x, y, w, h, color.NRGBA{ R: 255, G: 255, B: 255, A: 100 }, true)
		if i <= g.CurrentPage {
			vector.FillRect(screen, x, y, w, h, color.NRGBA{ R: 255, G: 255, B: 255, A: 255 }, true)
		}
			
		vector.StrokeRect(screen, x - 2, y - 2, w + 2, h + 2, 2, color.NRGBA{ R: 0, G: 0, B: 0, A: 255 }, true)
	}
}

func (g *Game) DrawPages(screen *ebiten.Image) {
	for i, cImage := range g.Images {
		op := &ebiten.DrawImageOptions{}
		translateX, translateY := g.GetPageTransformDiff(i) 
		var scale float64 = 1

		imageWidth := float64(cImage.Bounds().Dx())
		if g.ScreenWidth < imageWidth {
			scale = g.ScreenWidth / imageWidth 
		}
		
		scale *= g.GetPageScale(i)
		op.GeoM.Scale(scale, scale)
		
		currentPageXOffset := (float64(i) - g.VisualPage) * g.ScreenWidth
		op.GeoM.Translate(translateX + currentPageXOffset, translateY)

		clipRect := image.Rect(int(currentPageXOffset), 0, int(currentPageXOffset + g.ScreenWidth), int(g.ScreenHeight))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)
		clipped.DrawImage(cImage, op)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.CurrentScreen {
		case ChapterScreen: {
			g.DrawPages(screen)
			g.DrawPagination(screen)	
		}	
	}
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

func (g *Game) FetchChapter() (error, MangedexChapterResult) {
	var result MangedexChapterResult

	url := "https://api.mangadex.org/at-home/server/2230afe5-c254-425a-8e78-8d31011a915e"
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

	g.PageTransform = make([]PageTransform, len(result.Chapter.Data))
	g.PaginationPageHeight = make([]PaginationPageHeight, len(result.Chapter.Data))
	for i := range g.PageTransform {
    	g.PageTransform[i].Scale = 1.0
	}

	for _, chapterData := range result.Chapter.Data {
		img, _ := LoadImageFromUrl(result.BaseUrl + "/data/" + result.Chapter.Hash + "/" + chapterData)
		g.Images = append(g.Images, img)
	}
	// img, _ := LoadImageFromUrl("https://uploads.mangadex.org/data/3303dd03ac8d27452cce3f2a882e94b2/2-2a5e95dfec7f15cd01f9a63835be18a22fb77a10fd2d62858c7dcbb6e6c622f9.png")
	// game.Images = append(game.Images, img)

	return nil, result 
 }

func (g *Game) Fetch() {
	switch g.CurrentScreen {
		case ChapterScreen: {
			g.FetchChapter()	
		}	
	}	
}

func main() {
	g := Game{
		CurrentScreen: ChapterScreen,
	}

	g.Fetch()

	ebiten.SetWindowSize(800, 1200)
	ebiten.SetWindowTitle("Hello, World!")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
