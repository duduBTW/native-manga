package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"time"

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

const (
	CONFIG_DIR       = "native-manga"
	CONFIG_FILE_NAME = "settings.json"
)

type Screen int

const (
	MangaScreen Screen = iota
	ChapterScreen
	BrowseScreen
	LoginScreen
	HomeScreen
)

type ClickableRegion struct {
	Bounds  Bounds
	Id      string
	OnClick func()
	OnHover func()
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
	GameLogin

	FontTitle   *text.GoTextFace
	FontCaption *text.GoTextFace
	FontBody    text.Face
	FontBodySM  *text.GoTextFace

	SelectedElID string

	Auth MangadexAuth

	ClickableRegions []ClickableRegion
	Inputs           [](*InputOptions)
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
			g.BrowseCoverUpdate()
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
	case LoginScreen:
		{
			g.LoginUpdate()
		}
	}

	//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
	//	_mouseX, _mouseY := ebiten.CursorPosition()
	//	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	//	for _, region := range g.ClickableRegions {
	//		if region.Bounds.Contains(mouseX, mouseY) {
	//			region.OnClick()
	//		}
	//	}
	//}

	g.InputUpdate()

	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	for _, region := range g.ClickableRegions {
		if region.Bounds.Contains(mouseX, mouseY) {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				region.OnClick()
			} else if region.OnHover != nil {
				region.OnHover()
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.ClickableRegions = []ClickableRegion{}
	g.Inputs = [](*InputOptions){}

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
	case LoginScreen:
		{
			g.LoginDraw(screen)
		}
	}
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

	g.FontCaption = &text.GoTextFace{
		Source: titleTextFaceSource,
		Size:   21,
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

func (g *Game) Authenticate() {
	auth, err := GetConfig()
	if err != nil {
		return
	}

	if auth.AccessToken == "" || auth.RefreshToken == "" {
		return
	}

	if auth.ValidUntil.Before(time.Now()) {
		auth, err = MangadexRefresh(auth.RefreshToken, context.Background())
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			WriteConfig(auth)
		}()
	}

	g.Auth = auth
	g.CurrentScreen = HomeScreen 
}

func main() {
	g := Game{
		CurrentScreen: LoginScreen,
	}
	g.Authenticate()

	if err := g.LoadFonts(); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(800, 1200)
	ebiten.SetWindowTitle("Manga")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}

// Config
func GetConfigDir() string {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to find user config dir: %v", err)
	}

	appDir := filepath.Join(baseDir, CONFIG_DIR)
	filePath := filepath.Join(appDir, CONFIG_FILE_NAME)

	err = os.MkdirAll(appDir, 0o755)
	if err != nil {
		log.Fatalf("Failed to create app directory: %v", err)
	}

	return filePath
}

func GetConfig() (MangadexAuth, error) {
	var result MangadexAuth

	configBytes, err := os.ReadFile(GetConfigDir())
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(configBytes, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func WriteConfig(data MangadexAuth) error {
	jsonAuth, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigDir(), jsonAuth, 0o644)
}
