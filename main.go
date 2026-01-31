package main

import (
	"SNES_emulator/internal/config"
	"SNES_emulator/soc"
	"SNES_emulator/ui"
	"log"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	config := config.New()
	if config.IsPProfEnabled() {
		f, err := os.Create(config.GetPProfPath())
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	fb := ui.NewFramebuffer()
	display := ui.NewEmulatorDisplay(fb)

	soc := soc.NewSoC(config.GetRomPath(), fb, display.Controller0, display.Controller1)
	defer soc.Cartridge.SaveSramToFile() //save Sram(if exists) on exit

	ebiten.SetWindowTitle(soc.Cartridge.GetRomName())

	go soc.Run()
	ebiten.RunGame(display)
}
