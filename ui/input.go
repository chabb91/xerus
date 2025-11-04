package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type SNESController struct {
	Up      bool
	Down    bool
	Left    bool
	Right   bool
	AButton bool
	BButton bool
	Start   bool
	Select  bool
}

func (c *SNESController) MapInputToController() {
	c.Up = ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp)
	c.Down = ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown)
	c.Left = ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft)
	c.Right = ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight)

	c.AButton = ebiten.IsKeyPressed(ebiten.KeyK)
	c.BButton = ebiten.IsKeyPressed(ebiten.KeyJ)
	c.Start = ebiten.IsKeyPressed(ebiten.KeyEnter)
	c.Select = ebiten.IsKeyPressed(ebiten.KeySpace)

	if len(ebiten.GamepadIDs()) > 0 {
		gamepadID := ebiten.GamepadIDs()[0]

		c.Up = c.Up || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton3))
		c.Down = c.Down || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton4))

		c.AButton = c.AButton || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton1))
		c.BButton = c.BButton || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton2))
	}
}
