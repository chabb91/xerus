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
	XButton bool
	YButton bool
	Start   bool
	Select  bool
	LButton bool
	RButton bool
}

func (c *SNESController) Latch() uint16 {
	c.Up = ebiten.IsKeyPressed(ebiten.KeyUp)
	c.Down = ebiten.IsKeyPressed(ebiten.KeyDown)
	c.Left = ebiten.IsKeyPressed(ebiten.KeyLeft)
	c.Right = ebiten.IsKeyPressed(ebiten.KeyRight)

	c.AButton = ebiten.IsKeyPressed(ebiten.KeyA)
	c.BButton = ebiten.IsKeyPressed(ebiten.KeyS)
	c.XButton = ebiten.IsKeyPressed(ebiten.KeyW)
	c.YButton = ebiten.IsKeyPressed(ebiten.KeyE)

	c.LButton = ebiten.IsKeyPressed(ebiten.KeyZ)
	c.RButton = ebiten.IsKeyPressed(ebiten.KeyX)

	c.Start = ebiten.IsKeyPressed(ebiten.KeyEnter)
	c.Select = ebiten.IsKeyPressed(ebiten.KeySpace)

	/*if len(ebiten.GamepadIDs()) > 0 {
		gamepadID := ebiten.GamepadIDs()[0]

		c.Up = c.Up || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton3))
		c.Down = c.Down || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton4))

		c.AButton = c.AButton || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton1))
		c.BButton = c.BButton || ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(ebiten.GamepadButton2))
	}*/
	return c.GetControllerState()
}
func (c *SNESController) GetControllerState() uint16 {
	var state uint16 = 0

	if c.BButton {
		state |= 1
	}
	if c.YButton {
		state |= 1 << 1
	}
	if c.Select {
		state |= 1 << 2
	}
	if c.Start {
		state |= 1 << 3
	}
	if c.Up {
		state |= 1 << 4
	}
	if c.Down {
		state |= 1 << 5
	}
	if c.Left {
		state |= 1 << 6
	}
	if c.Right {
		state |= 1 << 7
	}
	if c.AButton {
		state |= 1 << 8
	}
	if c.XButton {
		state |= 1 << 9
	}
	if c.LButton {
		state |= 1 << 10
	}
	if c.RButton {
		state |= 1 << 11
	}

	return state
}
