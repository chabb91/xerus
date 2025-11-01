package main

import (
	"SNES_emulator/soc"
	"SNES_emulator/ui"
	"log"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	f, err := os.Create("/home/chabb/Documents/gopp/cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	ebiten.SetWindowTitle("SNES Emulator")
	ebiten.SetWindowSize(ui.DefaultWidth*ui.ScalingFactor, ui.DefaultHeight*ui.ScalingFactor)

	fb := ui.NewFramebuffer()
	display := ui.NewEmulatorDisplay(fb)

	soc := soc.NewSoC(fb)
	go soc.Run()
	ebiten.RunGame(display)
}
