package main

import (
	"bytes"
	_ "embed"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
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
)

type ClickableRegion struct {
	Bounds  Bounds
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

	GameManga
	GameChapter

	FontTitle *text.GoTextFace
	FontBody  text.Face

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
	case MangaScreen:
		{
			g.MangaChapterPageUpdate()
			g.UpdateMangaAnimation()
		}
	case ChapterScreen:
		{
			if g.ChapterCount() == 0 {
				return nil
			}

			g.ChapterImagesUpdate()
			g.ChapterPaginationUpdate()
			g.ChapterPageUpdate()
			g.UpdateChapterAnimation()
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.CurrentScreen {
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
	mangaId := "28b5d037-175d-4119-96f8-e860e408ebe9"
	mangaResult, err := FetchManga(mangaId)
	if err != nil {
		return err
	}

	for _, title := range mangaResult.Data.Attributes.Title {
		g.MangaTitle = title
		break
	}

	for _, description := range mangaResult.Data.Attributes.Description {
		g.MangaDescription = description
		break
	}

	imageCoverArtUrl, err := mangaResult.CoverArtImageUrl()
	if err != nil {
		return err
	}

	imgCoverArt, err := LoadImageFromUrl(imageCoverArtUrl)
	if err != nil {
		return err
	}

	g.MangaCoverArtImage = ebiten.NewImageFromImage(imgCoverArt)

	mangaChaptersResult, err := FetchMangaChapters(mangaId)
	if err != nil {
		return err
	}

	g.MangaChapterData = mangaChaptersResult.Data
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
	return nil
}

func main() {
	g := Game{
		CurrentScreen: MangaScreen,
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
