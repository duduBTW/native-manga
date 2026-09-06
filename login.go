package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GameLogin struct {
	LoginUsername string
	LoginPassword string
}

func (g *Game) LoginUpdate() {}

const LOGIN_LINE_HEIGHT = 32

func (g *Game) LoginHandleUsernameChange(value string) {
	g.LoginUsername = value
}

func (g *Game) LoginHandlePasswordChange(value string) {
	g.LoginPassword = value
}

func (g *Game) LoginFormDraw(screen *ebiten.Image, bounds Bounds) {
	vector.FillRect(screen, 0, 0, float32(g.ScreenWidth), float32(g.ScreenHeight), color.White, true)

	totalLines := int(g.ScreenHeight / LOGIN_LINE_HEIGHT)
	middlePoint := int(math.Floor(float64(totalLines) / 2))
	totalElements := 7
	startingPoint := middlePoint - totalElements
	loginLabel := "LOGIN"
	bodyWidth, bodyHeight := text.Measure(loginLabel, g.FontBodySM, 0)

	for i := 1; i < totalLines; i++ {
		y := float32(i * LOGIN_LINE_HEIGHT)
		vector.StrokeLine(screen, float32(bounds.X), y, float32(bounds.X+bounds.W), y, 1, color.Black, true)

		iop := &InputOptions{}
		iop.Bounds = Bounds{X: bounds.X, Y: float64(y - 28), W: bounds.W, H: 28}
		if i == startingPoint {
			op := &text.DrawOptions{}
			op.ColorScale.ScaleWithColor(color.Black)
			op.GeoM.Translate(bounds.X+12, float64(y)-bodyHeight-4)
			text.Draw(screen, "Username", g.FontBodySM, op)
		}

		if i == startingPoint+1 {
			iop.Placeholder = "xxTestxx"
			iop.ID = "login-username"
			iop.Value = g.LoginUsername
			iop.OnChange = g.LoginHandleUsernameChange
			g.DrawInput(screen, iop)
		}

		if i == startingPoint+3 {
			op := &text.DrawOptions{}
			op.ColorScale.ScaleWithColor(color.Black)
			op.GeoM.Translate(bounds.X+12, float64(y)-bodyHeight-4)
			text.Draw(screen, "Password", g.FontBodySM, op)
		}

		if i == startingPoint+4 {
			iop.Placeholder = "****"
			iop.ID = "login-password"
			iop.OnChange = g.LoginHandlePasswordChange

			censoredValue := ""
			for i := 0; i < len(g.LoginPassword); i++ {
				censoredValue += "*"
			}

			iop.Value = censoredValue
			g.DrawInput(screen, iop)
		}

		if i == startingPoint+6 {
			op := &text.DrawOptions{}
			op.ColorScale.ScaleWithColor(color.White)
			op.GeoM.Translate(bounds.X+12, float64(y)-bodyHeight-8)

			btnY := y - LOGIN_LINE_HEIGHT
			btnW := bodyWidth + 24
			vector.FillRect(screen,
				float32(bounds.X), btnY, float32(btnW), float32(LOGIN_LINE_HEIGHT),
				color.Black, false)

			text.Draw(screen, "LOGIN", g.FontBodySM, op)

			g.ClickableRegions = append(g.ClickableRegions, ClickableRegion{
				Bounds: Bounds{X: bounds.X, Y: float64(btnY), W: btnW, H: LOGIN_LINE_HEIGHT},
				OnClick: func() {
					//
				},
			})
		}
	}
}

func (g *Game) LoginDraw(screen *ebiten.Image) {
	g.LoginFormDraw(screen, Bounds{0, 0, g.ScreenWidth, g.ScreenHeight})
}
