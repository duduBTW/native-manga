package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
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
}

func (g *Game) BrowseFetch() {
	if g.BrowseFetchCancel != nil {
		g.BrowseFetchCancel()
		g.BrowseFetchCancel = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.BrowseFetchCancel = cancel

	go func() {
		result, err := FetchPopularNewTitles(ctx, g.BrowseSearchValue)
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

func (g *Game) BrowseUpdate() {
	_, scrollY := ebiten.Wheel()

	if scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.BrowseCurrentPage++
	}

	if scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.BrowseCurrentPage--
	}

	initialValue := g.BrowseSearchValue

	g.BrowseSearchValue += string(ebiten.AppendInputChars(nil))
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(g.BrowseSearchValue) > 0 {
			valueRune := []rune(g.BrowseSearchValue)
			g.BrowseSearchValue = string(valueRune[:len(valueRune)-1])
		}
	}

	if initialValue != g.BrowseSearchValue {
		g.BrowseSearchValueChanged = time.Now()
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
}

func (g *Game) BrowseHandleMangarClick(manga MangadexMangaData) {
	g.CurrentScreen = MangaScreen
	ctx, cancel := context.WithCancel(context.Background())
	g.MangaFetchCancel = cancel

	go func() {
		mangaResult, err := FetchManga(manga.Id, ctx)
		if err != nil {
			return
		}

		g.MangaID = manga.Id
		for _, title := range mangaResult.Data.Attributes.Title {
			g.MangaTitle = title
			break
		}

		for _, description := range mangaResult.Data.Attributes.Description {
			g.MangaDescription = description
			break
		}

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

func (g *Game) DrawBrowseMangaItem(screen *ebiten.Image, manga MangadexMangaData, bounds Bounds) {
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
	})
}

func (g *Game) DrawBrowseMangaGrid(screen *ebiten.Image, mangas []MangadexMangaData, bounds Bounds, itemWidth, itemHeight float64) {
	row := 0.0
	col := 0.0
	for _, manga := range mangas {
		x := bounds.X + (itemWidth * col)
		y := bounds.Y + (itemHeight * row)

		g.DrawBrowseMangaItem(
			screen,
			manga,
			Bounds{W: itemWidth, H: itemHeight, X: x, Y: y},
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

func (g *Game) DrawInput(screen *ebiten.Image, bounds Bounds) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(bounds.X+12, bounds.Y+8)

	textColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	value := g.BrowseSearchValue
	if value == "" {
		value = "Type to search..."
		textColor.A = 100
	}

	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, value, g.FontBodySM, op)
	y := float32(bounds.Y + bounds.H)
	vector.StrokeLine(screen, float32(bounds.X), y, float32(bounds.X+bounds.W), y, 2, color.Black, true)
}

func (g *Game) DrawBrowse(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)
	halfScreen := g.ScreenWidth / 2
	width := 0.0
	if g.IsBrowseFullWidth() {
		width = g.ScreenWidth
	} else {
		width = halfScreen
	}

	paddingTop := 32.0
	itemsPerRow := 4.0
	height := g.ScreenHeight - paddingTop
	itemWidth := width / itemsPerRow
	itemHeight := itemWidth * (732.0 / 512.0)

	rowsPerPage := math.Floor(height / itemHeight)
	itemsPerPage := rowsPerPage * itemsPerRow
	mangaCount := float64(len(g.BrowseData))
	pageCount := math.Ceil(mangaCount / itemsPerPage)

	inputWidth := 280.0
	g.DrawInput(screen, Bounds{X: g.ScreenWidth - inputWidth, Y: 0, W: inputWidth, H: 28})

	for i := 0.0; i < pageCount; i++ {
		start := itemsPerPage * i
		end := math.Min(start+itemsPerPage, mangaCount)

		xOffset := width * float64(i-g.BrowseVisualPage)
		yOffset := math.Max(32, (height-(rowsPerPage*itemHeight))/2)

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
		)
	}
}
