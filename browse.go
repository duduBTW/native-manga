package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameBrowse struct {
	BrowseIsInit                   bool
	BrowseFetchCancel              context.CancelFunc
	BrowseCoverArtFetchImageResult chan FetchImageResult
	BrowseMangaImages              map[string](*ebiten.Image)
	BrowseData                     []MangadexMangaData
	BrowseFetchImageResult         chan FetchImageResult

	BrowseSearchValueChanged time.Time
	BrowseCurrentPage        int
	BrowseVisualPage         float64
	BrowseSearchValue        string
	BrowseSelectedMangaRow   AnimatedProp
	BrowseSelectedMangaCol   AnimatedProp
}

func (g *Game) BrowseFetch() {
	if g.BrowseFetchCancel != nil {
		g.BrowseFetchCancel()
		g.BrowseFetchCancel = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.BrowseFetchCancel = cancel

	go func() {
		result, err := FetchPopularNewTitles(ctx, g.BrowseSearchValue, g.Auth.AccessToken)
		if err != nil {
			return
		}

		g.BrowseData = result.Data
		g.BrowseMangaImages = make(map[string](*ebiten.Image), len(g.BrowseData))
		g.BrowseCoverArtFetchImageResult = make(chan FetchImageResult, len(g.BrowseData))
		for _, manga := range g.BrowseData {
			go func() {
				imageCoverArtURL, err := manga.CoverArtImageUrl()
				if err != nil {
					return
				}

				imgCoverArt, err := LoadImageFromUrl(imageCoverArtURL+".512.jpg", ctx)
				if err != nil {
					return
				}

				g.BrowseCoverArtFetchImageResult <- FetchImageResult{Image: imgCoverArt, Err: err, Id: manga.Id}
			}()
		}
	}()
}

func (g *Game) BrowseSelectedMangaIndex() int {
	gridNumber := math.Floor(float64(g.BrowseSelectedMangaCol.Value) / g.BrowseMangasPerRow())
	restNumber := g.BrowseSelectedMangaCol.Value % int(g.BrowseMangasPerRow())
	rowNumber := g.BrowseMangasPerRow() * float64(g.BrowseSelectedMangaRow.Value)

	_, itemsPerPage := g.BrowseMangaGridCounds()
	index := gridNumber * itemsPerPage
	index += float64(restNumber)
	index += rowNumber
	return int(index)
}

func (g *Game) BrowseUpdate() {
	_, scrollY := ebiten.Wheel()

	if scrollY < 0 {
		g.BrowseCurrentPage++
	}
	if scrollY > 0 {
		g.BrowseCurrentPage--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.BrowseSelectedMangaRow.Value = int(math.Max(0, float64(g.BrowseSelectedMangaRow.Value-1)))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.BrowseSelectedMangaRow.Value = int(math.Min(g.BrowseMangaRowsPerPage()-1, float64(g.BrowseSelectedMangaRow.Value+1)))
	}
	if scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.BrowseSelectedMangaCol.Value++

		if g.BrowseSelectedMangaCol.Value%int(g.BrowseMangasPerRow()) == 0 {
			g.BrowseCurrentPage++
		}
	}

	if scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		if g.BrowseSelectedMangaCol.Value%int(g.BrowseMangasPerRow()) == 0 {
			g.BrowseCurrentPage--
		}
		g.BrowseSelectedMangaCol.Value--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.BrowseHandleMangarClick(g.BrowseData[g.BrowseSelectedMangaIndex()])
	}

	if !g.BrowseIsInit {
		g.BrowseIsInit = true
		g.BrowseFetch()
	}

	if !g.BrowseSearchValueChanged.IsZero() && time.Now().After(g.BrowseSearchValueChanged.Add(400*time.Millisecond)) {
		g.BrowseSearchValueChanged = time.Time{}
		g.BrowseFetch()
	}
}

func (g *Game) BrowseCoverUpdate() {
	if g.BrowseCoverArtFetchImageResult == nil {
		return
	}

	select {
	case res := <-g.BrowseCoverArtFetchImageResult:
		{
			g.BrowseMangaImages[res.Id] = ebiten.NewImageFromImage(res.Image)
		}
	default:
		{
		}
	}
}

func (g *Game) BrowseUpdateAnimation() {
	g.BrowseVisualPage += (float64(g.BrowseCurrentPage) - g.BrowseVisualPage) * 0.1
	g.BrowseSelectedMangaCol.UpdateBy(0.16)
	g.BrowseSelectedMangaRow.UpdateBy(0.2)
}

func (g *Game) BrowseHandleSearchInputChange(value string) {
	initialValue := g.BrowseSearchValue

	g.BrowseSearchValue = value

	if initialValue != g.BrowseSearchValue {
		g.BrowseSearchValueChanged = time.Now()
	}
}

func (g *Game) BrowseHandleMangarClick(manga MangadexMangaData) {
	g.CurrentScreen = MangaScreen
	ctx, cancel := context.WithCancel(context.Background())
	g.MangaFetchCancel = cancel

	for _, title := range manga.Attributes.Title {
		g.MangaTitle = title
		break
	}

	for _, description := range manga.Attributes.Description {
		g.MangaDescription = description
		break
	}

	go func() {
		mangaResult, err := FetchManga(manga.Id, ctx)
		if err != nil {
			return
		}

		g.MangaID = manga.Id

		imageCoverArtURL, err := mangaResult.CoverArtImageUrl()
		if err != nil {
			fmt.Println(err)
			return
		}

		imgCoverArt, err := LoadImageFromUrl(imageCoverArtURL, ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		g.MangaCoverArtFetchImageResult = make(chan FetchImageResult, 1)
		g.MangaCoverArtFetchImageResult <- FetchImageResult{
			Image: imgCoverArt,
			Err:   err,
			Id:    manga.Id,
		}
	}()

	go func() {
		mangaChaptersResult, err := FetchMangaChapters(manga.Id, ctx)
		if err != nil {
			return
		}

		g.MangaChapterData = mangaChaptersResult.Data
	}()
}

func (g *Game) BrowseMangasPerRow() float64 {
	return 4.0
}

func (g *Game) BrowseMangaDimensions() (float64, float64, float64, float64) {
	halfScreen := g.ScreenWidth / 2
	width := 0.0
	if g.IsBrowseFullWidth() {
		width = g.ScreenWidth
	} else {
		width = halfScreen
	}

	paddingTop := 32.0
	height := g.ScreenHeight - paddingTop
	itemWidth := width / g.BrowseMangasPerRow()
	itemHeight := itemWidth * (732.0 / 512.0)
	return itemWidth, itemHeight, width, height
}

func (g *Game) BrowseMangaCount() float64 {
	return float64(len(g.BrowseData))
}

func (g *Game) BrowseMangaRowsPerPage() float64 {
	_, itemHeight, _, height := g.BrowseMangaDimensions()
	return math.Floor(height / itemHeight)
}

func (g *Game) BrowseMangaGridCounds() (float64, float64) {
	itemsPerPage := g.BrowseMangaRowsPerPage() * g.BrowseMangasPerRow()
	mangaCount := float64(len(g.BrowseData))
	pageCount := math.Ceil(mangaCount / itemsPerPage)

	return pageCount, itemsPerPage
}

func drawTextWithShadow(dst *ebiten.Image, face text.Face, str string, x, y float64) {
	// pass 1: shadow — same string, offset down a couple px, dark + semi-transparent
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(x, y+2)
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255})
	shadowOp.ColorScale.ScaleAlpha(0.5) // tweak to taste
	text.Draw(dst, str, face, shadowOp)

	// pass 2: real text on top, no offset
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(dst, str, face, op)
}

func (g *Game) DrawBrowseMangaItem(screen *ebiten.Image, manga MangadexMangaData, bounds Bounds, row, col, index float64) {
	img, ok := g.BrowseMangaImages[manga.Id]
	if !ok {
		vector.StrokeRect(screen, float32(bounds.X), float32(bounds.Y), float32(bounds.W), float32(bounds.H), 1, color.Black, true)
		return
	}

	imageWidth, imageHeight := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())

	scaleW := bounds.W / imageWidth
	scaleH := bounds.H / imageHeight

	scale := math.Max(scaleW, scaleH)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)

	offsetX := (bounds.W - (imageWidth * scale)) / 2
	offsetY := (bounds.H - (imageHeight * scale)) / 2
	op.GeoM.Translate(bounds.X+offsetX, bounds.Y+offsetY)

	clipRect := image.Rect(
		int(bounds.X),
		int(bounds.Y),
		int(bounds.X+bounds.W),
		int(bounds.Y+bounds.H),
	)
	clipped := screen.SubImage(clipRect).(*ebiten.Image)
	clipped.DrawImage(img, op)

	g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
		Bounds: Bounds{X: bounds.X, Y: bounds.Y, W: bounds.W, H: bounds.H},
		OnClick: func() {
			g.BrowseHandleMangarClick(manga)
		},
		OnHover: func() {
			g.BrowseSelectedMangaRow.Value = int(row)
			g.BrowseSelectedMangaCol.Value = int(col + (g.BrowseMangasPerRow() * index))
		},
	})
}

func (g *Game) DrawBrowseMangaGrid(screen *ebiten.Image, mangas []MangadexMangaData, bounds Bounds, itemWidth, itemHeight, index float64) {
	row := 0.0
	col := 0.0
	for _, manga := range mangas {
		x := bounds.X + (itemWidth * col)
		y := bounds.Y + (itemHeight * row)

		g.DrawBrowseMangaItem(
			screen,
			manga,
			Bounds{W: itemWidth, H: itemHeight, X: x, Y: y},
			row, col, index,
		)

		if col >= 3 {
			col = 0
			row++
		} else {
			col++
		}
	}
}

func (g *Game) IsBrowseFullWidth() bool {
	return g.ScreenWidth <= 800
}

func (g *Game) DrawBrowseHighlight(screen *ebiten.Image) {
	if len(g.BrowseData) == 0 {
		return
	}

	itemWidth, itemHeight, width, height := g.BrowseMangaDimensions()
	x := (width * (g.BrowseVisualPage * -1))
	x += g.BrowseSelectedMangaCol.Visual * itemWidth

	y := math.Max(32, (height-(g.BrowseMangaRowsPerPage()*itemHeight))/2)
	y += g.BrowseSelectedMangaRow.Visual * itemHeight
	vector.StrokeRect(screen, float32(x-2), float32(y-2), float32(itemWidth+4), float32(itemHeight+4), 4, color.Black, true)

	manga := g.BrowseData[g.BrowseSelectedMangaIndex()]

	title := ""
	for _, t := range manga.Attributes.Title {
		title = t
		break
	}
	if title == "" {
		return
	}

	textPadding := 10.0
	lines := WrapText(
		title,
		g.FontCaption,
		itemWidth-textPadding,
	)

	_, captionHeight := text.Measure("A", g.FontCaption, 0)
	titleHeight := captionHeight*float64(len(lines)) + 10
	vector.FillRect(screen, float32(x-1), float32(y+itemHeight-titleHeight), float32(itemWidth+2), float32(titleHeight), color.NRGBA{R: 0, G: 0, B: 0, A: 255}, false)

	textOffsetTop := captionHeight
	for _, line := range slices.Backward(lines) {
		drawTextWithShadow(screen, g.FontCaption, line, x+5, y+itemHeight-textOffsetTop-5)
		textOffsetTop += captionHeight
	}
}

func (g *Game) DrawBrowse(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)

	itemWidth, itemHeight, width, height := g.BrowseMangaDimensions()
	pageCount, itemsPerPage := g.BrowseMangaGridCounds()
	inputWidth := 280.0
	iop := &InputOptions{}
	iop.Bounds = Bounds{X: g.ScreenWidth - inputWidth, Y: 0, W: inputWidth, H: 28}
	iop.Value = g.BrowseSearchValue
	iop.Placeholder = "Type to search..."
	iop.ID = "browse-search"
	iop.OnChange = g.BrowseHandleSearchInputChange
	g.DrawInput(screen, iop)

	for i := 0.0; i < pageCount; i++ {
		start := itemsPerPage * i
		end := math.Min(start+itemsPerPage, g.BrowseMangaCount())

		xOffset := width * float64(i-g.BrowseVisualPage)
		yOffset := math.Max(32, (height-(g.BrowseMangaRowsPerPage()*itemHeight))/2)

		g.DrawBrowseMangaGrid(
			screen, g.BrowseData[int(start):int(end)],
			Bounds{
				X: xOffset,
				Y: yOffset,
				W: width,
				H: height,
			},
			itemWidth,
			itemHeight,
			i,
		)
	}

	g.DrawBrowseHighlight(screen)
}
