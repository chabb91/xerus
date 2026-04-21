package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/chabb91/xerus/internal/config"
	"github.com/chabb91/xerus/soc"
	"github.com/chabb91/xerus/ui"

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
	display := ui.NewEmulatorDisplay(fb, config)

	soc := soc.NewSoC(config, fb, display.Controller0, display.Controller1)
	defer soc.Cartridge.SaveSramToFile() //save Sram(if exists) on exit

	ebiten.SetWindowTitle(soc.Cartridge.GetRomName())
	go soc.Run()
	ui.GetEmulatorAudio().Play(soc.Spu.Dsp.Buffer)
	ebiten.RunGame(display)
}
