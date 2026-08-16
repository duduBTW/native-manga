package main

import (
	"context"
	"errors"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameManga struct {
	MangaCoverArtImage            *ebiten.Image
	MangaCoverArtFetchImageResult chan FetchImageResult
	MangaTitle                    string
	MangaDescription              string
	MangaChapterData              []MangadexMangaChapterData
	MangaFetchCancel              context.CancelFunc

	SelectedChapterId string

	MangaCurrentPage int
	MangaVisualPage  float64

	MangaLastX, MangaLastY float64
}

func (g *GameManga) Clean() {
	g.MangaCurrentPage = 0
	g.MangaVisualPage = 0
	g.MangaTitle = ""
	g.MangaDescription = ""
	g.MangaChapterData = nil
	g.SelectedChapterId = ""
	g.MangaLastX = 0
	g.MangaLastY = 0
	if g.MangaCoverArtImage != nil {
		g.MangaCoverArtImage.Deallocate()
	}

	g.MangaCoverArtImage = nil
}

const mangaChapterPagePadding float64 = 80

func (g *Game) MangaChapterIsKeyboardInteractable() bool {
	return g.MangaCurrentPage > 1 || !g.IsMangaFullWidth() && g.MangaCurrentPage > 0
}

func (g *Game) UpdateMangaAnimation() {
	g.MangaVisualPage += (float64(g.MangaCurrentPage) - g.MangaVisualPage) * 0.1
}

func (g *Game) MangaUpdate() {
	_, scrollY := ebiten.Wheel()
	baseBounds, _ := g.MangaChapterPageBaseBounds(0, 0)
	if scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) && g.MangaCurrentPage < int(len(g.MangaChapterPageCount(baseBounds))+1) {
		g.MangaCurrentPage++
	}

	if (scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)) && g.MangaCurrentPage > 0 {
		g.MangaCurrentPage--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.CurrentScreen = BrowseScreen
		g.MangaFetchCancel()
		g.GameManga.Clean()
		return
	}

	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	hasMoved := mouseX != g.MangaLastX || mouseY != g.MangaLastY
	if hasMoved {
		g.MangaLastX = mouseX
		g.MangaLastY = mouseY
		for _, region := range g.ClickableRegions {
			if region.Bounds.Contains(mouseX, mouseY) {
				g.SelectedChapterId = region.Id
			}
		}
	}

	selectChapterDirection := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		selectChapterDirection = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		selectChapterDirection = -1
	}
	if selectChapterDirection != 0 && g.MangaChapterIsKeyboardInteractable() {
		hasChangedSelection := false
		for i, chapter := range g.MangaChapterData {
			if g.SelectedChapterId != chapter.Id {
				continue
			}

			newSelectedIndex := i + selectChapterDirection
			if newSelectedIndex < 0 || newSelectedIndex > len(g.MangaChapterData)-1 {
				break
			}
			hasChangedSelection = true
			g.SelectedChapterId = g.MangaChapterData[newSelectedIndex].Id
			break
		}

		if !hasChangedSelection {
			g.SelectedChapterId = g.MangaChapterData[0].Id
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && g.SelectedChapterId != "" && g.MangaChapterIsKeyboardInteractable() {
		for _, region := range g.ClickableRegions {
			if region.Id != g.SelectedChapterId {
				continue
			}

			region.OnClick()
			break
		}
	}
}

func (g *Game) MangaCoverUpdate() {
	if g.MangaCoverArtFetchImageResult == nil {
		return
	}

	select {
	case res := <-g.MangaCoverArtFetchImageResult:
		{
			g.MangaCoverArtImage = ebiten.NewImageFromImage(res.Image)
		}
	default:
		{
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

func (g *Game) MangaChapterPageBaseBounds(x, y float64) (Bounds, float64) {
	padding := mangaChapterPagePadding
	halfScreen := g.ScreenWidth / 2
	width := 0.0
	if g.IsMangaFullWidth() {
		width = g.ScreenWidth
	} else {
		width = halfScreen
	}

	return Bounds{X: x + padding, Y: y + padding, W: width - (padding * 2), H: g.MangaChapterPageHeight()}, width
}

func (g *Game) MangaChapterPageCount(bounds Bounds) [][]MangadexMangaChapterData {
	var result [][]MangadexMangaChapterData
	currentPageItems := []MangadexMangaChapterData{}
	currentPageAccHeight := 0.0

	_, bodyHeight := text.Measure("A", g.FontBody, 0)
	for _, chapter := range g.MangaChapterData {
		lines := WrapText(
			chapter.Attributes.Chapter+chapter.Attributes.Title,
			g.FontBody,
			bounds.W-70,
		)

		chapterHeight := bodyHeight*float64(len(lines)) + 12
		if currentPageAccHeight+chapterHeight > bounds.H {
			currentPageAccHeight = 0
			result = append(result, currentPageItems)
			currentPageItems = []MangadexMangaChapterData{}
		}

		currentPageItems = append(currentPageItems, chapter)
		currentPageAccHeight += chapterHeight
	}

	if len(currentPageItems) > 0 {
		result = append(result, currentPageItems)
	}
	return result
}

func (g *Game) DrawMangaCover(screen *ebiten.Image, bounds Bounds) {
	if g.MangaCoverArtImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear

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

	ctx, cancel := context.WithCancel(context.Background())
	g.FetchImageCancel = cancel

	go func() {
		result, err := FetchChapter(chapter.Id, ctx)
		if err != nil && errors.Is(err, context.Canceled) {
			return
		}

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
				if g.CurrentScreen != ChapterScreen {
					return
				}

				pageURL := result.BaseUrl + "/data/" + result.Chapter.Hash + "/" + chapterData
				img, err := LoadImageFromUrl(pageURL, ctx)
				if err != nil {
					// TODO(Dudu): Show retry button if the error is not a context cancelation.
					return
				}

				g.FetchImageResult <- FetchImageResult{Image: img, Err: err, Id: chapterData}
			}()
		}
	}()
}

func (g *Game) DrawMangaChapterPage(screen *ebiten.Image, chapters []MangadexMangaChapterData, bounds Bounds) {
	currentHeight := bounds.Y
	for _, chapter := range chapters {
		ID := chapter.Id
		isHovered := g.SelectedChapterId == ID
		textColor := color.Black
		initialHeight := currentHeight
		if isHovered {
			textColor = color.White
		}

		lines := WrapText(
			chapter.Attributes.Title,
			g.FontBody,
			bounds.W-70,
		)

		_, bodyHeight := text.Measure("A", g.FontBody, 0)
		if isHovered {
			vector.FillRect(screen, float32(bounds.X), float32(initialHeight), float32(bounds.W), float32(bodyHeight)*float32(len(lines)), color.Black, true)
		}

		chapterNumOp := &text.DrawOptions{}
		chapterNumOp.ColorScale.ScaleWithColor(textColor)
		chapterNumOp.GeoM.Translate(bounds.X, currentHeight)
		text.Draw(screen, chapter.Attributes.Chapter, g.FontBody, chapterNumOp)

		for _, line := range lines {
			chapterTextOp := &text.DrawOptions{}
			chapterTextOp.ColorScale.ScaleWithColor(textColor)

			chapterTextOp.GeoM.Translate(bounds.X+70, currentHeight)
			currentHeight += bodyHeight
			//chapterTextOp.LineSpacing = g.FontBody.Metrics().HLineGap

			text.Draw(screen, line, g.FontBody, chapterTextOp)
		}

		g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
			Id:     ID,
			Bounds: Bounds{X: bounds.X, Y: initialHeight, W: bounds.W, H: currentHeight - initialHeight},
			OnClick: func() {
				g.MangaHandleChapterClick(chapter)
			},
		})

		currentHeight += 12
	}
}

func (g *Game) DrawMangaChapters(screen *ebiten.Image, x, y float64) {
	padding := mangaChapterPagePadding

	baseBounds, width := g.MangaChapterPageBaseBounds(x, y)
	for i, chapters := range g.MangaChapterPageCount(baseBounds) {
		g.DrawMangaChapterPage(
			screen,
			chapters,
			Bounds{X: (width * float64(i)) + x + padding, Y: baseBounds.Y, W: baseBounds.W, H: baseBounds.H},
		)
	}
}

func (g *Game) IsMangaFullWidth() bool {
	return g.ScreenWidth <= 800
}

func (g *Game) DrawManga(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)

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
