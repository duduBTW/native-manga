package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameBrowse struct {
	BrowseMangaImages      map[string](*ebiten.Image)
	BrowseData             []MangadexMangaData
	BrowseFetchImageResult chan FetchImageResult

	BrowseCurrentPage int
	BrowseVisualPage  float64
}

func (g *Game) BrowseUpdate() {
	_, scrollY := ebiten.Wheel()

	if scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.BrowseCurrentPage++
	}

	if scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.BrowseCurrentPage--
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
			fmt.Println(err)
			return
		}
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

func (g *Game) DrawBrowse(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)
	halfScreen := g.ScreenWidth / 2
	width := 0.0
	if g.IsBrowseFullWidth() {
		width = g.ScreenWidth
	} else {
		width = halfScreen
	}

	height := g.ScreenHeight
	itemWidth := width / 4
	itemHeight := itemWidth * (732.0 / 512.0)

	rowsPerPage := math.Floor(height / itemHeight)
	itemsPerPage := rowsPerPage * 4
	mangaCount := float64(len(g.BrowseData))
	pageCount := math.Ceil(mangaCount / itemsPerPage)

	for i := 0.0; i < pageCount; i++ {
		start := itemsPerPage * i
		end := math.Min(start+itemsPerPage, mangaCount)

		xOffset := width * float64(i-g.BrowseVisualPage)
		yOffset := (height - (rowsPerPage * itemHeight)) / 2

		g.DrawBrowseMangaGrid(screen, g.BrowseData[int(start):int(end)],
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
