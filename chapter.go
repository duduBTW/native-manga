package main

import (
	"errors"
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type ChapterPageTransformType int

const (
	Last ChapterPageTransformType = iota
	Initial
	Current
)

const CurrentPage = -1

type PageAxisTransformLast struct {
	Real   float64
	Visual float64
}

type ChapterPageAxisTransform struct {
	Last    PageAxisTransformLast
	Initial float64
	Current float64
}

func (t *ChapterPageAxisTransform) ChapterCalculateLastOffset() float64 {
	return t.Last.Real + t.Current - t.Initial
}

type ChapterPageTransform struct {
	X, Y  ChapterPageAxisTransform
	Scale float64
}

type PaginationPageHeight struct {
	Current float32
	Visual  float32
}

type FetchImageResult struct {
	Image image.Image
	Err   error
	Id    string
}

type GameChapter struct {
	PageImages           map[string](*ebiten.Image)
	PageTransform        []ChapterPageTransform
	CurrentPage          int
	VisualPage           float64
	PaginationPageHeight []PaginationPageHeight
	IsLoadingChapter     bool
	ChapterLoadErr       error
	ChapterData          MangedexChapter
	FetchImageResult     chan FetchImageResult
}

func (g *GameChapter) Clean() {
	g.CurrentPage = 0
	g.VisualPage = 0
	g.ChapterData = MangedexChapter{}
	for id, img := range g.PageImages {
		if img != nil {
			img.Deallocate()
		}
		delete(g.PageImages, id)
	}
}

func (g *Game) ChapterGetCurrentPageTransform(pageIndex int) *ChapterPageTransform {
	if pageIndex == CurrentPage {
		pageIndex = g.CurrentPage
	}
	return &g.PageTransform[pageIndex]
}

func (g *Game) ChapterGetPageTransformDiff(pageIndex int) (float64, float64) {
	lastX, lastY := g.ChapterGetPageTransform(Last, pageIndex)
	initialX, initialY := g.ChapterGetPageTransform(Initial, pageIndex)
	currentX, currentY := g.ChapterGetPageTransform(Current, pageIndex)
	return lastX + currentX - initialX, lastY + currentY - initialY
}

func (g *Game) ChapterPageCalculateLast(pageIndex int) (float64, float64) {
	transform := g.ChapterGetCurrentPageTransform(pageIndex)
	return transform.X.ChapterCalculateLastOffset(), transform.Y.ChapterCalculateLastOffset()
}

func (g *Game) ChapterGetPageTransform(prop ChapterPageTransformType, pageIndex int) (float64, float64) {
	transform := g.ChapterGetCurrentPageTransform(pageIndex)
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

func (g *Game) ChapterSetPageTransform(prop ChapterPageTransformType, x, y float64, pageIndex int) {
	transform := g.ChapterGetCurrentPageTransform(pageIndex)
	switch prop {
	case Last:
		{
			transform.X.Last.Real = x
			transform.Y.Last.Real = y

			transform.X.Last.Visual = x
			transform.Y.Last.Visual = y
		}
	case Initial:
		{
			transform.X.Initial = x
			transform.Y.Initial = y
		}
	case Current:
		{
			transform.X.Current = x
			transform.Y.Current = y
		}
	}
}

func (g *Game) ChapterGetPageScale(pageIndex int) float64 {
	transform := g.ChapterGetCurrentPageTransform(pageIndex)
	return transform.Scale
}

func (g *Game) ChapterSetPageScale(value float64) {
	transform := g.ChapterGetCurrentPageTransform(CurrentPage)
	transform.Scale = value
}

func (g *Game) ChapterNavigateTo(target int) error {
	if target < 0 || target+1 > g.ChapterCount() {
		return errors.New("Target out of bounds")
	}

	g.CurrentPage = target
	return nil
}

func (g *Game) ChapterPreviousPage() {
	g.ChapterNavigateTo(g.CurrentPage - 1)
}

func (g *Game) ChapterNextPage() {
	g.ChapterNavigateTo(g.CurrentPage + 1)
}

func (g *Game) UpdateChapterAnimation() {
	g.VisualPage += (float64(g.CurrentPage) - g.VisualPage) * 0.1
	for i := range g.ChapterData.Data {
		pageHeight := g.PaginationPageHeight[i]
		g.PaginationPageHeight[i].Visual += (pageHeight.Current - pageHeight.Visual) * 0.12

		pageYLast := g.PageTransform[i].Y.Last
		g.PageTransform[i].Y.Last.Visual += (pageYLast.Real - pageYLast.Visual) * 0.14
	}
}

func (g *Game) ChapterPaginationUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)

	if mouseY < g.ScreenHeight-80 {
		for i := range g.ChapterData.Data {
			g.PaginationPageHeight[i].Current = 0
		}

		return
	}

	for i := 0; i < g.ChapterCount(); i++ {
		var defaultHeight float32 = 10
		aroundHoveredHeight := defaultHeight + 10
		hoveredHeight := aroundHoveredHeight + 12

		g.PaginationPageHeight[i].Current = defaultHeight
		if mouseY > g.ScreenHeight-float64(hoveredHeight) {
			size, x := g.ChapterPageItemWidth(i)

			if mouseX > float64(x) && mouseX < float64(x+size) {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					g.ChapterNavigateTo(i)
				}

				isFirst := i == 0
				if !isFirst {
					g.PaginationPageHeight[i-1].Current = aroundHoveredHeight
				}

				g.PaginationPageHeight[i].Current = hoveredHeight

				isLast := g.ChapterCount() == i+1
				if !isLast {
					g.PaginationPageHeight[i+1].Current = aroundHoveredHeight
					i++
				}
			}
		}
	}
}

func (g *Game) ChapterPageUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	scrollX, scrollY := ebiten.Wheel()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.ChapterSetPageTransform(Initial, mouseX, mouseY, CurrentPage)
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.ChapterSetPageTransform(Current, mouseX, mouseY, CurrentPage)
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		hasPageMouseMoved := g.ChapterHasPageMouseMoved()
		if hasPageMouseMoved {
			lastX, lastY := g.ChapterPageCalculateLast(CurrentPage)
			g.ChapterSetPageTransform(Last, lastX, lastY, CurrentPage)
			g.ChapterSetPageTransform(Current, 0, 0, CurrentPage)
			g.ChapterSetPageTransform(Initial, 0, 0, CurrentPage)
		} else if mouseY < g.ScreenHeight-32 {
			center := g.ScreenWidth / 2
			if mouseX < center {
				g.ChapterPreviousPage()
			} else {
				g.ChapterNextPage()
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.CurrentScreen = MangaScreen
		g.GameChapter.Clean()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.ChapterNextPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.ChapterPreviousPage()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
		oldScale := g.ChapterGetPageScale(CurrentPage)
		newScale := math.Max(0, oldScale+(scrollY/10))

		if oldScale > 0 && newScale > 0 {
			scaleFactor := newScale / oldScale

			lastX, lastY := g.ChapterGetPageTransform(Last, CurrentPage)

			newLastX := mouseX - (mouseX-lastX)*scaleFactor
			newLastY := mouseY - (mouseY-lastY)*scaleFactor

			g.ChapterSetPageTransform(Last, newLastX, newLastY, CurrentPage)
			g.ChapterSetPageScale(newScale)
		}
	} else {
		var multiplier float64 = 42
		var keyboardMultiplier float64 = 32
		lastY := g.PageTransform[g.CurrentPage].Y.Last.Real
		lastX := g.PageTransform[g.CurrentPage].X.Last.Real

		yScrollAmount := lastY + (scrollY * multiplier)
		xScrollAmount := lastX + (scrollX * multiplier)
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			yScrollAmount -= keyboardMultiplier
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			yScrollAmount += keyboardMultiplier
		}
		g.PageTransform[g.CurrentPage].Y.Last.Real = yScrollAmount
		g.PageTransform[g.CurrentPage].X.Last.Real = xScrollAmount
	}
}

func (g *Game) ChapterImagesUpdate() {
	if g.FetchImageResult == nil {
		return
	}

	select {
	case res := <-g.FetchImageResult:
		{
			g.PageImages[res.Id] = ebiten.NewImageFromImage(res.Image)

			position := -1
			for i, data := range g.ChapterData.Data {
				if data == res.Id {
					position = i
					break
				}
			}

			if position != -1 {
				g.ChapterCenterPage(g.PageImages[res.Id], position)
			}
		}
	default:
		{
		}
	}
}

func (g *Game) ChapterPageItemWidth(i int) (float32, float32) {
	var gap float32 = 6
	totalItems := float32(g.ChapterCount())
	if totalItems > 30 {
		gap = 4
	}
	width := (float32(g.ScreenWidth) - (gap * totalItems)) / totalItems
	x := (width * float32(i)) + (gap * float32(i)) + 2
	return width, x
}

func (g *Game) DrawChapterPagination(screen *ebiten.Image) {
	for i := range g.ChapterData.Data {
		w, x := g.ChapterPageItemWidth(i)
		h := float32(g.PaginationPageHeight[i].Visual)
		y := float32(g.ScreenHeight) - h
		vector.FillRect(screen, x, y, w, h, color.NRGBA{R: 255, G: 255, B: 255, A: 155}, true)
		if i <= g.CurrentPage {
			vector.FillRect(screen, x, y, w, h, color.NRGBA{R: 185, G: 217, B: 230, A: 255}, true)
		}

		pageNumber := strconv.Itoa(i + 1)
		textW, textH := text.Measure(pageNumber, g.FontBodySM, 0)
		if float64(h) > textH+8 && textW < float64(w) {
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+((w-float32(textW))/2)), float64(y)+5)
			op.ColorScale.ScaleWithColor(color.Black)

			text.Draw(screen, pageNumber, g.FontBodySM, op)
		}

		vector.StrokeRect(screen, x, y+1, w, h, 1, color.Black, true)
	}
}

func (g *Game) DrawChapterPages(screen *ebiten.Image) {
	for i, chapterId := range g.ChapterData.Data {
		cImage, ok := g.PageImages[chapterId]
		if !ok {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear

		translateX, translateY := g.ChapterGetPageTransformDiff(i)
		var scale float64 = 1

		imageWidth := float64(cImage.Bounds().Dx())
		if g.ScreenWidth < imageWidth {
			scale = g.ScreenWidth / imageWidth
		}

		scale *= g.ChapterGetPageScale(i)
		op.GeoM.Scale(scale, scale)

		currentPageXOffset := (float64(i) - g.VisualPage) * g.ScreenWidth
		op.GeoM.Translate(translateX+currentPageXOffset, translateY)

		clipRect := image.Rect(int(currentPageXOffset), 0, int(currentPageXOffset+g.ScreenWidth), int(g.ScreenHeight))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)
		clipped.DrawImage(cImage, op)
	}
}

func (g *Game) ChapterHasPageMouseMoved() bool {
	initialX, initialY := g.ChapterGetPageTransform(Initial, CurrentPage)
	currentX, currentY := g.ChapterGetPageTransform(Current, CurrentPage)
	return initialX != currentX || initialY != currentY
}

func (g *Game) ChapterCenterPage(img *ebiten.Image, position int) {
	imgWidth, imgHeight := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	g.PageTransform[position].Scale = 1
	g.ChapterSetPageTransform(Last, 0, 0, position)

	if imgWidth < g.ScreenWidth {
		g.ChapterSetPageTransform(Last, (g.ScreenWidth-imgWidth)/2, 0, position)
	}

	scale := g.ScreenWidth / imgWidth
	scaledImageHeight := imgHeight * scale
	if scaledImageHeight < g.ScreenHeight {
		g.ChapterSetPageTransform(Last, 0, (g.ScreenHeight-scaledImageHeight)/2, position)
	}
}

func (g *Game) ChapterCenterPages() {
	for i, cID := range g.ChapterData.Data {
		img, ok := g.PageImages[cID]
		if !ok {
			continue
		}

		g.ChapterCenterPage(img, i)
	}
}

func (g *Game) ChapterCount() int {
	return len(g.ChapterData.Data)
}
