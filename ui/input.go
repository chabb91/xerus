package ui

import (
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type SnesInput interface {
	UpdateControllerState()
	Latch() uint16
}

// Empty interface for unattached controllers
type NullInput struct {
}

func (c *NullInput) UpdateControllerState() {

}
func (c *NullInput) Latch() uint16 {
	return 0
}

type SNESKeyboardInput struct {
	buttons atomic.Uint32
}

func (c *SNESKeyboardInput) Latch() uint16 {
	return uint16(c.buttons.Load())
}

func (c *SNESKeyboardInput) UpdateControllerState() {
	var state uint16 = 0

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		state |= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		state |= 1 << 1
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		state |= 1 << 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		state |= 1 << 3
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		state |= 1 << 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		state |= 1 << 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		state |= 1 << 6
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		state |= 1 << 7
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		state |= 1 << 8
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		state |= 1 << 9
	}
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		state |= 1 << 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		state |= 1 << 11
	}

	c.buttons.Store(uint32(state))
}

type SNESControllerInput struct {
	buttons atomic.Uint32

	controllerId   ebiten.GamepadID
	isDisconnected bool

	gamepadIDsBuf []ebiten.GamepadID
}

func NewSnesControllerInput(id ebiten.GamepadID) *SNESControllerInput {
	return &SNESControllerInput{
		controllerId: id}
}

func (c *SNESControllerInput) Latch() uint16 {
	return uint16(c.buttons.Load())
}

func (c *SNESControllerInput) UpdateControllerState() {

	c.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(c.gamepadIDsBuf[:0])
	for _, id := range c.gamepadIDsBuf {
		if id == c.controllerId {
			c.isDisconnected = false
		}
	}
	if inpututil.IsGamepadJustDisconnected(c.controllerId) {
		c.isDisconnected = true
	}
	if c.isDisconnected {
		c.buttons.Store(0)
		return
	}

	var state uint16 = 0

	//B, Y, Select, Start, Up, Down, Left, Right, A, X, L, R, 0, 0, 0, 0
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton0) {
		state |= 1
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton2) {
		state |= 1 << 1
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton6) {
		state |= 1 << 2
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton7) {
		state |= 1 << 3
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton11) {
		state |= 1 << 4
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton13) {
		state |= 1 << 5
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton14) {
		state |= 1 << 6
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton12) {
		state |= 1 << 7
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton1) {
		state |= 1 << 8
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton3) {
		state |= 1 << 9
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton4) {
		state |= 1 << 10
	}
	if ebiten.IsGamepadButtonPressed(c.controllerId, ebiten.GamepadButton5) {
		state |= 1 << 11
	}

	c.buttons.Store(uint32(state))
}
