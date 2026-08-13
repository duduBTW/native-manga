package main

import (
	"bytes"
	"context"
	_ "embed"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed assets/BebasNeue-Regular.ttf
var titleFontTTF []byte

//go:embed assets/animeace2_reg.ttf
var bodyRegularFontTTF []byte

//go:embed assets/MochiyPopOne-Regular.ttf
var bodyRegularJpFontTTF []byte

type Screen int

const (
	MangaScreen Screen = iota
	ChapterScreen
	BrowseScreen
)

type ClickableRegion struct {
	Bounds  Bounds
	Id      string
	OnClick func()
}

type Bounds struct {
	X, Y, W, H float64
}

func (b Bounds) Contains(x, y float64) bool {
	return x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H
}

type Game struct {
	ScreenHeight float64
	ScreenWidth  float64

	CurrentScreen Screen

	GameBrowse
	GameManga
	GameChapter

	FontTitle  *text.GoTextFace
	FontBody   text.Face
	FontBodySM *text.GoTextFace

	ClickableRegions []ClickableRegion
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	hasResized := g.ScreenWidth != float64(outsideWidth) || g.ScreenHeight != float64(outsideHeight)

	g.ScreenWidth = float64(outsideWidth)
	g.ScreenHeight = float64(outsideHeight)

	if hasResized && g.CurrentScreen == ChapterScreen {
		g.ChapterCenterPages()
	}

	return outsideWidth, outsideHeight
}

func (g *Game) Update() error {
	switch g.CurrentScreen {
	case BrowseScreen:
		{
			g.BrowseUpdate()
			g.BrowseUpdateAnimation()
		}
	case MangaScreen:
		{
			g.MangaCoverUpdate()
			g.MangaUpdate()
			g.UpdateMangaAnimation()
		}
	case ChapterScreen:
		{
			g.ChapterUpdate()

			if g.ChapterCount() == 0 {
				return nil
			}

			g.ChapterImagesUpdate()
			g.ChapterPaginationUpdate()
			g.ChapterPageUpdate()
			g.UpdateChapterAnimation()
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		_mouseX, _mouseY := ebiten.CursorPosition()
		mouseX, mouseY := float64(_mouseX), float64(_mouseY)
		for _, region := range g.ClickableRegions {
			if region.Bounds.Contains(mouseX, mouseY) {
				region.OnClick()
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.ClickableRegions = []ClickableRegion{}

	switch g.CurrentScreen {
	case BrowseScreen:
		{
			g.DrawBrowse(screen)
		}
	case MangaScreen:
		{
			g.DrawManga(screen)
		}
	case ChapterScreen:
		{
			if g.ChapterCount() == 0 {
				return
			}

			g.DrawChapterPages(screen)
			g.DrawChapterPagination(screen)
		}
	}
}

func (g *Game) Fetch() error {
	result, err := FetchPopularNewTitles()
	if err != nil {
		return err
	}

	g.BrowseData = result.Data
	g.BrowseMangaImages = make(map[string](*ebiten.Image), len(g.BrowseData))

	for _, manga := range g.BrowseData {
		imageCoverArtURL, err := manga.CoverArtImageUrl()
		if err != nil {
			return err
		}

		imgCoverArt, err := LoadImageFromUrl(imageCoverArtURL+".512.jpg", context.Background())
		if err != nil {
			return err
		}

		g.BrowseMangaImages[manga.Id] = ebiten.NewImageFromImage(imgCoverArt)
	}

	return nil
}

func (g *Game) LoadFonts() error {
	titleTextFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(titleFontTTF))
	if err != nil {
		return err
	}

	g.FontTitle = &text.GoTextFace{
		Source: titleTextFaceSource,
		Size:   40,
	}

	bodyTextFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(bodyRegularFontTTF))
	if err != nil {
		return err
	}

	bodyJpTextFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(bodyRegularJpFontTTF))
	if err != nil {
		return err
	}

	const bodySize = 18
	bodyFont := &text.GoTextFace{Source: bodyTextFaceSource, Size: bodySize}
	bodyJpFont := &text.GoTextFace{Source: bodyJpTextFaceSource, Size: bodySize}
	bodyMulti, err := text.NewMultiFace(bodyFont, bodyJpFont)
	if err != nil {
		return err
	}

	g.FontBody = bodyMulti
	g.FontBodySM = &text.GoTextFace{
		Source: bodyTextFaceSource,
		Size:   12,
	}
	return nil
}

func main() {
	g := Game{
		CurrentScreen: BrowseScreen,
	}

	if err := g.LoadFonts(); err != nil {
		log.Fatal(err)
	}

	if err := g.Fetch(); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(800, 1200)
	ebiten.SetWindowTitle("Manga")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
