package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameBrowse struct {
	BrowseMangaImages      map[string](*ebiten.Image)
	BrowseData             []MangadexMangaData
	BrowseFetchImageResult chan FetchImageResult
}

func (g *Game) BrowseHandleMangarClick(manga MangadexMangaData) {
	g.CurrentScreen = MangaScreen

	go func() {
		mangaResult, err := FetchManga(manga.Id)
		if err != nil {
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

		imgCoverArt, err := LoadImageFromUrl(imageCoverArtURL)
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
		mangaChaptersResult, err := FetchMangaChapters(manga.Id)
		if err != nil {
			return
		}

		g.MangaChapterData = mangaChaptersResult.Data
	}()
}

func (g *Game) DrawBrowseMangaGrid(screen *ebiten.Image, mangas []MangadexMangaData, bounds Bounds) {
	itemWidth := bounds.W / 4
	itemHeight := itemWidth * (732.0 / 512.0)

	row := 0
	col := 0
	for _, manga := range mangas {
		img, ok := g.BrowseMangaImages[manga.Id]
		if !ok {
			continue
		}

		imageWidth, imageHeight := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())

		scaleW := itemWidth / imageWidth
		scaleH := itemHeight / imageHeight

		scale := math.Max(scaleW, scaleH)

		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear
		op.GeoM.Scale(scale, scale)

		x := bounds.X + (itemWidth * float64(col))
		y := bounds.Y + (itemHeight * float64(row))

		offsetX := (itemWidth - (imageWidth * scale)) / 2
		offsetY := (itemHeight - (imageHeight * scale)) / 2

		op.GeoM.Translate(x+offsetX, y+offsetY)

		clipRect := image.Rect(int(x), int(y), int(x+itemWidth), int(y+itemHeight))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)
		clipped.DrawImage(img, op)

		g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
			Bounds: Bounds{X: x, Y: y, W: itemWidth, H: itemHeight},
			OnClick: func() {
				g.BrowseHandleMangarClick(manga)
			},
		})

		if col >= 3 {
			col = 0
			row++
		} else {
			col++
		}
	}
}

func (g *Game) DrawBrowse(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)

	g.DrawBrowseMangaGrid(screen, g.BrowseData, Bounds{
		X: 0,
		Y: 0,
		W: g.ScreenWidth,
		H: g.ScreenHeight,
	})
}
