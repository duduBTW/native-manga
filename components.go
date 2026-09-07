package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type InputOptions struct {
	ID          string
	Bounds      Bounds
	Placeholder string
	Value       string
	OnChange    func(string)
	OnFocus     func()
}

func (g *Game) InputChange(iop *InputOptions) {
	value := iop.Value + string(ebiten.AppendInputChars(nil))
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(value) > 0 {
			valueRune := []rune(value)
			value = string(valueRune[:len(valueRune)-1])
		}
	}

	if iop.OnChange != nil {
		iop.OnChange(value)
	}
}

func (g *Game) InputUpdate() {
	hasInputBeenSelected := false
	hasClicked := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	_mouseX, _mouseY := ebiten.CursorPosition()
	mouseX, mouseY := float64(_mouseX), float64(_mouseY)

	var selectedIop *InputOptions
	for _, iop := range g.Inputs {
		if iop.Bounds.Contains(mouseX, mouseY) && hasClicked {
			hasInputBeenSelected = true
			g.SelectedElID = iop.ID

			if iop.OnFocus != nil {
				iop.OnFocus()
			}
			break
		}

		if iop.ID == g.SelectedElID {
			selectedIop = iop
		}
	}

	if !hasInputBeenSelected && hasClicked {
		selectedIop = nil
		g.SelectedElID = ""
		return
	}

	if selectedIop != nil && g.SelectedElID != "" {
		g.InputChange(selectedIop)
	}
}

func (g *Game) DrawInput(screen *ebiten.Image, iop *InputOptions) {
	g.Inputs = append(g.Inputs, iop)

	value := iop.Value
	bounds := iop.Bounds

	op := &text.DrawOptions{}
	op.GeoM.Translate(bounds.X+12, bounds.Y+8)

	strokeColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	if g.SelectedElID != "" && g.SelectedElID == iop.ID {
		strokeColor.B = 255
	}

	textColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	if value == "" {
		value = iop.Placeholder
		textColor.A = 100
	}

	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, value, g.FontBodySM, op)
	y := float32(bounds.Y + bounds.H)
	vector.StrokeLine(screen, float32(bounds.X), y, float32(bounds.X+bounds.W), y, 2, strokeColor, true)
}

const (
	BUTTON_HEIGHT  = 32
	BUTTON_PADDING = 24
)

type ButtonVariant int

const (
	ButtonPrimary ButtonVariant = iota
	ButtonSeconday
)

var storedButtonWidth float64

type ButtonOptions struct {
	TextContent string
	OnClick     func()
	Variant     ButtonVariant
	Bounds
}

func (g *Game) DrawButton(screen *ebiten.Image, bop *ButtonOptions) {
	bodyWidth, bodyHeight := text.Measure(bop.TextContent, g.FontBodySM, 0)

	bounds := bop.Bounds

	op := &text.DrawOptions{}
	op.ColorScale.ScaleWithColor(color.White)
	op.GeoM.Translate(bounds.X+12, bounds.Y+(bodyHeight/2))

	btnW := bodyWidth + BUTTON_PADDING
	storedButtonWidth = btnW

	bgColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	if bop.Variant == ButtonSeconday {
		bgColor.A = 170
	}
	vector.FillRect(screen,
		float32(bounds.X), float32(bounds.Y), float32(btnW), float32(LOGIN_LINE_HEIGHT),
		bgColor, false)

	text.Draw(screen, bop.TextContent, g.FontBodySM, op)

	g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
		Bounds:  Bounds{X: bounds.X, Y: bounds.Y, W: btnW, H: BUTTON_HEIGHT},
		OnClick: bop.OnClick,
	})
}

func ButtonWidth() float64 {
	return storedButtonWidth
}
