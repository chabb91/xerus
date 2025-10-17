package main

import (
	"SNES_emulator/soc"
	"SNES_emulator/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	fb := ui.NewFramebuffer()
	display := ui.NewEmulatorDisplay(fb)

	soc := soc.NewSoC(fb)
	go soc.Run()
	ebiten.RunGame(display)
}
