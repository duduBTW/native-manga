package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"net/http"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

type Transform int

const (
	Last Transform = iota
	Initial
	Current
)

const CurrentPage = -1

type PageAxisTransformLast struct {
	Real   float64
	Visual float64
}

type PageAxisTransform struct {
	Last    PageAxisTransformLast
	Initial float64
	Current float64
}

func (t *PageAxisTransform) CalculateLast() float64 {
	return t.Last.Real + t.Current - t.Initial
}

type PageTransform struct {
	X, Y  PageAxisTransform
	Scale float64
}

type PaginationPageHeight struct {
	Current float32
	Visual  float32
}

type Bounds struct {
	X, Y, W, H float64
}

func (b Bounds) Contains(x, y float64) bool {
	return x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H
}

type ClickableRegion struct {
	Bounds  Bounds
	OnClick func()
}

type FetchImageResult struct {
	Image image.Image
	Err   error
	Id    string
}

type Game struct {
	CoverArtImage *ebiten.Image

	PageImages           map[string](*ebiten.Image)
	PageTransform        []PageTransform
	CurrentPage          int
	VisualPage           float64
	PaginationPageHeight []PaginationPageHeight
	IsLoadingChapter     bool
	ChapterLoadErr       error
	ChapterData          MangedexChapter
	FetchImageResult     chan FetchImageResult

	ScreenHeight float64
	ScreenWidth  float64

	CurrentScreen Screen

	MangaTitle       string
	MangaDescription string
	MangaChapterData []MangadexMangaChapterData
	CurrentMangaPage int
	VisualMangaPage  float64

	FontTitle *text.GoTextFace
	FontBody  text.Face

	ClickableRegions []ClickableRegion
}

func (g *Game) GetCurrentPageTransform(pageIndex int) *PageTransform {
	if pageIndex == CurrentPage {
		pageIndex = g.CurrentPage
	}
	return &g.PageTransform[pageIndex]
}

func (g *Game) HasPageMouseMoved() bool {
	initialX, initialY := g.GetPageTransform(Initial, CurrentPage)
	currentX, currentY := g.GetPageTransform(Current, CurrentPage)
	return initialX != currentX || initialY != currentY
}

func (g *Game) GetPageTransformDiff(pageIndex int) (float64, float64) {
	lastX, lastY := g.GetPageTransform(Last, pageIndex)
	initialX, initialY := g.GetPageTransform(Initial, pageIndex)
	currentX, currentY := g.GetPageTransform(Current, pageIndex)
	return lastX + currentX - initialX, lastY + currentY - initialY
}

func (g *Game) PageCalculateLast(pageIndex int) (float64, float64) {
	transform := g.GetCurrentPageTransform(pageIndex)
	return transform.X.CalculateLast(), transform.Y.CalculateLast()
}

func (g *Game) GetPageTransform(prop Transform, pageIndex int) (float64, float64) {
	transform := g.GetCurrentPageTransform(pageIndex)
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

func (g *Game) SetPageTransform(prop Transform, x, y float64, pageIndex int) {
	transform := g.GetCurrentPageTransform(pageIndex)
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

func (g *Game) GetPageScale(pageIndex int) float64 {
	transform := g.GetCurrentPageTransform(pageIndex)
	return transform.Scale
}

func (g *Game) SetPageScale(value float64) {
	transform := g.GetCurrentPageTransform(CurrentPage)
	transform.Scale = value
}

func (g *Game) CenterPage(img *ebiten.Image, position int) {
	imgWidth, imgHeight := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	g.PageTransform[position].Scale = 1
	if imgWidth < g.ScreenWidth {
		g.SetPageTransform(Last, (g.ScreenWidth-imgWidth)/2, 0, position)
	}

	if imgHeight < g.ScreenHeight {
		scale := g.ScreenWidth / imgWidth
		g.SetPageTransform(Last, 0, (g.ScreenHeight-imgHeight*scale)/2, position)
	}
}

func (g *Game) CenterPages() {
	for i, cId := range g.ChapterData.Data {
		img, ok := g.PageImages[cId]
		if !ok {
			continue
		}

		g.CenterPage(img, i)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	hasResized := g.ScreenWidth != float64(outsideWidth) || g.ScreenHeight != float64(outsideHeight)

	g.ScreenWidth = float64(outsideWidth)
	g.ScreenHeight = float64(outsideHeight)

	if hasResized {
		g.CenterPages()
	}

	return outsideWidth, outsideHeight
}

func (g *Game) NavigateTo(target int) error {
	if target < 0 || target+1 > len(g.ChapterData.Data) {
		return errors.New("Target out of bounds")
	}

	g.CurrentPage = target
	return nil
}

func (g *Game) PreviousPage() {
	g.NavigateTo(g.CurrentPage - 1)
}

func (g *Game) NextPage() {
	g.NavigateTo(g.CurrentPage + 1)
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

func (g *Game) UpdateMangaAnimation() {
	g.VisualMangaPage += (float64(g.CurrentMangaPage) - g.VisualMangaPage) * 0.1
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

	for i := 0; i < len(g.ChapterData.Data); i++ {
		var defaultHeight float32 = 10
		var aroundHoveredHeight float32 = defaultHeight + 10
		var hoveredHeight float32 = aroundHoveredHeight + 12

		g.PaginationPageHeight[i].Current = defaultHeight
		if mouseY > g.ScreenHeight-float64(hoveredHeight) {
			size, x := g.PageSize(i)

			if mouseX > float64(x) && mouseX < float64(x+size) {
				if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
					g.NavigateTo(i)
				}

				isFirst := i == 0
				if !isFirst {
					g.PaginationPageHeight[i-1].Current = aroundHoveredHeight
				}

				g.PaginationPageHeight[i].Current = hoveredHeight

				isLast := len(g.ChapterData.Data) == i+1
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
		g.SetPageTransform(Initial, mouseX, mouseY, CurrentPage)
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.SetPageTransform(Current, mouseX, mouseY, CurrentPage)
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		hasPageMouseMoved := g.HasPageMouseMoved()
		if hasPageMouseMoved {
			lastX, lastY := g.PageCalculateLast(CurrentPage)
			g.SetPageTransform(Last, lastX, lastY, CurrentPage)
			g.SetPageTransform(Current, 0, 0, CurrentPage)
			g.SetPageTransform(Initial, 0, 0, CurrentPage)
		} else {
			center := g.ScreenWidth / 2
			if mouseX < center {
				g.PreviousPage()
			} else {
				g.NextPage()
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.CurrentScreen = MangaScreen
		g.CurrentPage = 0
		g.VisualPage = 0
		for id, img := range g.PageImages {
			if img != nil {
				img.Dispose()
			}
			delete(g.PageImages, id)
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.NextPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.PreviousPage()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
		oldScale := g.GetPageScale(CurrentPage)
		newScale := math.Max(0, oldScale+(scrollY/10))

		if oldScale > 0 && newScale > 0 {
			scaleFactor := newScale / oldScale

			lastX, lastY := g.GetPageTransform(Last, CurrentPage)

			newLastX := mouseX - (mouseX-lastX)*scaleFactor
			newLastY := mouseY - (mouseY-lastY)*scaleFactor

			g.SetPageTransform(Last, newLastX, newLastY, CurrentPage)
			g.SetPageScale(newScale)
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

func (g *Game) MangaChapterPageUpdate() {
	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)
	_, scrollY := ebiten.Wheel()

	if (scrollY < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)) && g.CurrentMangaPage < int(g.MangaChapterPageCount()+1) {
		g.CurrentMangaPage++
	}

	if (scrollY > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)) && g.CurrentMangaPage > 0 {
		g.CurrentMangaPage--
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, region := range g.ClickableRegions {
			if region.Bounds.Contains(mouseX, mouseY) {
				region.OnClick()
			}
		}
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
				g.CenterPage(g.PageImages[res.Id], position)
			}
		}
	default:
		{
		}
	}
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
			if len(g.ChapterData.Data) == 0 {
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

func (g *Game) PageSize(i int) (float32, float32) {
	var gap float32 = 6
	totalItems := float32(len(g.ChapterData.Data))
	size := (float32(g.ScreenWidth) - (gap * totalItems)) / totalItems
	x := (size * float32(i)) + (gap * float32(i))
	return size, x
}

func (g *Game) DrawPagination(screen *ebiten.Image) {
	for i := range g.ChapterData.Data {
		w, x := g.PageSize(i)
		h := float32(g.PaginationPageHeight[i].Visual)
		y := float32(g.ScreenHeight) - h
		vector.FillRect(screen, x, y, w, h, color.NRGBA{R: 255, G: 255, B: 255, A: 50}, true)
		if i <= g.CurrentPage {
			vector.FillRect(screen, x, y, w, h, color.White, true)
		}

		vector.StrokeRect(screen, x, y+1, w, h, 1, color.Black, true)
	}
}

func (g *Game) DrawPages(screen *ebiten.Image) {
	for i, chapterId := range g.ChapterData.Data {
		cImage, ok := g.PageImages[chapterId]
		if !ok {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		translateX, translateY := g.GetPageTransformDiff(i)
		var scale float64 = 1

		imageWidth := float64(cImage.Bounds().Dx())
		if g.ScreenWidth < imageWidth {
			scale = g.ScreenWidth / imageWidth
		}

		scale *= g.GetPageScale(i)
		op.GeoM.Scale(scale, scale)

		currentPageXOffset := (float64(i) - g.VisualPage) * g.ScreenWidth
		op.GeoM.Translate(translateX+currentPageXOffset, translateY)

		clipRect := image.Rect(int(currentPageXOffset), 0, int(currentPageXOffset+g.ScreenWidth), int(g.ScreenHeight))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)
		clipped.DrawImage(cImage, op)
	}
}

func (g *Game) DrawMangaCover(screen *ebiten.Image, bounds Bounds) {
	op := &ebiten.DrawImageOptions{}

	imageOriginalWidth, imageOriginalHeight := float64(g.CoverArtImage.Bounds().Dx()), float64(g.CoverArtImage.Bounds().Dy())
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
	screen.DrawImage(g.CoverArtImage, op)
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
				g.CurrentScreen = ChapterScreen
				g.IsLoadingChapter = true
				g.ChapterLoadErr = nil

				go func() {
					result, err := FetchChapter(chapter.Id)
					g.ChapterLoadErr = err
					g.ChapterData = result.Chapter

					g.PageTransform = make([]PageTransform, len(result.Chapter.Data))
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
							img, err := LoadImageFromUrl(result.BaseUrl + "/data/" + result.Chapter.Hash + "/" + chapterData)
							g.FetchImageResult <- FetchImageResult{Image: img, Err: err, Id: chapterData}
						}()
					}
				}()
			},
		})
	}
}

func (g *Game) MangaChapterCount() float64 {
	return float64(len(g.MangaChapterData))
}

func (g *Game) ChapterCount() float64 {
	return float64(len(g.ChapterData.Data))
}

const mangaChapterPagePadding float64 = 80

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

func (g *Game) DrawMangaChapters(screen *ebiten.Image, x, y float64) {
	var padding float64 = mangaChapterPagePadding
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

	xOffset := (width * float64(g.VisualMangaPage)) * -1

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

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.CurrentScreen {
	case MangaScreen:
		{
			g.DrawManga(screen)
		}
	case ChapterScreen:
		{
			if len(g.ChapterData.Data) == 0 {
				return
			}

			g.DrawPages(screen)
			g.DrawPagination(screen)
		}
	}
}

var mangaId string = "28b5d037-175d-4119-96f8-e860e408ebe9"

func (g *Game) FetchManga() (error, MangadexManga) {
	var result MangadexManga

	url := "https://api.mangadex.org/manga/" + mangaId + "?includes[]=cover_art"
	res, err := http.Get(url)
	if err != nil {
		return err, result
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("Failed to fetch"), result
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return err, result
	}

	for _, title := range result.Data.Attributes.Title {
		g.MangaTitle = title
		break
	}

	for _, description := range result.Data.Attributes.Description {
		g.MangaDescription = description
		break
	}

	baseUrl := "https://mangadex.org/covers/"
	coverArtFileName := ""
	for _, relationship := range result.Data.Relationships {
		switch relationship.Type {
		case "cover_art":
			coverArtFileName = relationship.Attributes.FileName
		}
	}
	if coverArtFileName == "" {
		return errors.New("Cover art not found"), result
	}

	imgCoverArt, _ := LoadImageFromUrl(baseUrl + result.Data.Id + "/" + coverArtFileName)
	g.CoverArtImage = ebiten.NewImageFromImage(imgCoverArt)

	return err, result
}

func (g *Game) FetchMangaChapters() (error, MangadexMangaChapterResponse) {
	var result MangadexMangaChapterResponse

	url := "https://api.mangadex.org/manga/" + mangaId + "/feed?translatedLanguage[]=en&limit=96&includes[]=scanlation_group&includes[]=user&order[volume]=desc&order[chapter]=desc&offset=0&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=erotica&contentRating[]=pornographic&includeUnavailable=0"
	res, err := http.Get(url)
	if err != nil {
		return err, result
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("Failed to fetch"), result
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return err, result
	}

	g.MangaChapterData = result.Data

	return nil, result
}

func (g *Game) Fetch() error {
	switch g.CurrentScreen {
	case MangaScreen:
		{
			err, _ := g.FetchManga()
			if err != nil {
				return err
			}

			err, _ = g.FetchMangaChapters()
			return err
		}
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
	ebiten.SetWindowTitle("Hello, World!")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
