package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameManga struct {
	MangaCoverArtImage *ebiten.Image
	MangaTitle         string
	MangaDescription   string
	MangaChapterData   []MangadexMangaChapterData

	MangaCurrentPage int
	MangaVisualPage  float64
}

const mangaChapterPagePadding float64 = 80

func (g *Game) UpdateMangaAnimation() {
	g.MangaVisualPage += (float64(g.MangaCurrentPage) - g.MangaVisualPage) * 0.1
}

func (g *Game) MangaChapterPageUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	_, scrollY := ebiten.Wheel()

	if (scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)) && g.MangaCurrentPage < int(g.MangaChapterPageCount()+1) {
		g.MangaCurrentPage++
	}

	if (scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)) && g.MangaCurrentPage > 0 {
		g.MangaCurrentPage--
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, region := range g.ClickableRegions {
			if region.Bounds.Contains(mouseX, mouseY) {
				region.OnClick()
			}
		}
	}
}

func (g *Game) MangaChapterCount() float64 {
	return float64(len(g.MangaChapterData))
}

func (g *Game) MangaChapterPageHeight() float64 {
	return g.ScreenHeight - (mangaChapterPagePadding * 2)
}

func (g *Game) MangaChapterPageRowHeight() float64 {
	_, lineHeight := text.Measure("A", g.FontBody, 0)
	itemPadding := 10.0
	return lineHeight + itemPadding
}

func (g *Game) MangaChapterPerPage() float64 {
	return math.Floor(g.MangaChapterPageHeight() / g.MangaChapterPageRowHeight())
}

func (g *Game) MangaChapterPageCount() float64 {
	return math.Ceil(g.MangaChapterCount() / g.MangaChapterPerPage())
}

func (g *Game) DrawMangaCover(screen *ebiten.Image, bounds Bounds) {
	op := &ebiten.DrawImageOptions{}

	imageOriginalWidth, imageOriginalHeight := float64(g.MangaCoverArtImage.Bounds().Dx()), float64(g.MangaCoverArtImage.Bounds().Dy())
	var scale float64 = 1

	if imageOriginalWidth > bounds.W {
		scale = bounds.W / imageOriginalWidth
	} else if imageOriginalHeight > bounds.H {
		scale = bounds.H / imageOriginalHeight
	}
	scale = math.Min(1, scale)
	imageHeight := imageOriginalHeight * scale
	imageWidth := imageOriginalWidth * scale

	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(bounds.X, bounds.Y)
	op.GeoM.Translate(((imageWidth/2)-(bounds.W/2))*-1, ((imageHeight/2)-(bounds.H/2))*-1)
	screen.DrawImage(g.MangaCoverArtImage, op)
}

func (g *Game) DrawMangaDetails(screen *ebiten.Image, bounds Bounds) {
	titleOp := &text.DrawOptions{}
	titleOp.ColorScale.ScaleWithColor(color.Black)
	titleOp.GeoM.Translate(bounds.X, bounds.Y+bounds.H-g.FontTitle.Size)
	text.Draw(screen, g.MangaTitle, g.FontTitle, titleOp)

	lines := WrapText(g.MangaDescription, g.FontBody, bounds.W)

	_, bodyHeight := text.Measure("A", g.FontBody, 0)
	for i, line := range lines {
		descriptionOp := &text.DrawOptions{}
		descriptionOp.ColorScale.ScaleWithColor(color.NRGBA{R: 0, G: 0, B: 0, A: 180})
		descriptionOp.GeoM.Translate(bounds.X, bounds.Y+(bodyHeight*float64(i)))
		descriptionOp.LineSpacing = g.FontBody.Metrics().HLineGap
		text.Draw(screen, line, g.FontBody, descriptionOp)
	}
}

func (g *Game) MangaHandleChapterClick(chapter MangadexMangaChapterData) {
	g.CurrentScreen = ChapterScreen
	g.IsLoadingChapter = true
	g.ChapterLoadErr = nil

	go func() {
		result, err := FetchChapter(chapter.Id)
		g.ChapterLoadErr = err
		g.ChapterData = result.Chapter

		g.PageTransform = make([]ChapterPageTransform, len(result.Chapter.Data))
		g.PaginationPageHeight = make([]PaginationPageHeight, len(result.Chapter.Data))
		for i := range g.PageTransform {
			g.PageTransform[i].Scale = 1.0
		}
		g.FetchImageResult = make(chan FetchImageResult, len(result.Chapter.Data))
		g.PageImages = make(map[string](*ebiten.Image), len(result.Chapter.Data))
		g.CurrentPage = 0
		g.VisualPage = 0
	
		for _, chapterData := range result.Chapter.Data {
			go func() {
				pageURL := result.BaseUrl + "/data/" + result.Chapter.Hash + "/" + chapterData
				img, err := LoadImageFromUrl(pageURL)
				g.FetchImageResult <- FetchImageResult{Image: img, Err: err, Id: chapterData}
			}()
		}
	}()
}

func (g *Game) DrawMangaChapterPage(screen *ebiten.Image, chapters []MangadexMangaChapterData, bounds Bounds) {
	rowHeight := g.MangaChapterPageRowHeight()
	for i, chapter := range chapters {
		chapterTextOp := &text.DrawOptions{}
		chapterTextOp.ColorScale.ScaleWithColor(color.Black)
		x := bounds.X
		y := bounds.Y + (rowHeight * float64(i))
		chapterTextOp.GeoM.Translate(x, y)
		text.Draw(screen, "Ch. "+chapter.Attributes.Chapter+"  "+chapter.Attributes.Title, g.FontBody, chapterTextOp)

		g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
			Bounds: Bounds{X: x, Y: y, W: bounds.W, H: rowHeight},
			OnClick: func() {
				g.MangaHandleChapterClick(chapter)
			},
		})
	}
}

func (g *Game) DrawMangaChapters(screen *ebiten.Image, x, y float64) {
	padding := mangaChapterPagePadding
	g.ClickableRegions = []ClickableRegion{}

	for i := 0.0; i < g.MangaChapterPageCount(); i++ {
		start := g.MangaChapterPerPage() * i
		end := math.Min(start+g.MangaChapterPerPage(), g.MangaChapterCount())
		halfScreen := g.ScreenWidth / 2
		width := 0.0
		if g.IsMangaFullWidth() {
			width = g.ScreenWidth
		} else {
			width = halfScreen
		}

		g.DrawMangaChapterPage(
			screen,
			g.MangaChapterData[int(start):int(end)],
			Bounds{X: (width * i) + x + padding, Y: y + padding, W: width - (padding * 2), H: g.MangaChapterPageHeight()},
		)
	}
}

func (g *Game) IsMangaFullWidth() bool {
	return g.ScreenWidth <= 800
}

func (g *Game) DrawManga(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.NRGBA{R: 255, G: 255, B: 255, A: 255}, true)

	halfScreen := g.ScreenWidth / 2
	width := 0.0
	if g.IsMangaFullWidth() {
		width = g.ScreenWidth
	} else {
		width = halfScreen
	}

	xOffset := (width * float64(g.MangaVisualPage)) * -1

	detailsBounds := Bounds{X: xOffset + 80, Y: 80, W: width - 160, H: g.ScreenHeight - 160}
	g.DrawMangaDetails(screen, detailsBounds)

	coverBounds := Bounds{X: xOffset + halfScreen, Y: 0, W: width, H: g.ScreenHeight}
	if g.IsMangaFullWidth() {
		coverBounds.X = g.ScreenWidth + xOffset
	}
	g.DrawMangaCover(screen, coverBounds)

	chaptersXOffset := g.ScreenWidth + xOffset
	if g.IsMangaFullWidth() {
		chaptersXOffset += g.ScreenWidth
	}
	g.DrawMangaChapters(screen, chaptersXOffset, 0)
}
