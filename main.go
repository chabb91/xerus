package main

import (
	"SNES_emulator/soc"
	"fmt"
)

func main() {
	soc := soc.NewSoC()
	var cnt uint64

	cpuTickRate := 6
	dmaTickRate := 8

	var dmaOn bool
	for range 560000 {
		cnt++
		soc.MulDiv.StepCycle()
		if soc.Dma.Mdmaen != 0 && cnt == uint64(dmaTickRate) {
			if !dmaOn {
				fmt.Println("doing little dma on channels ", soc.Dma.Mdmaen)
				dmaOn = true
			}
			soc.Dma.Step()
			cnt = 0
			continue

		}
		if soc.Dma.Mdmaen == 0 && cnt == uint64(cpuTickRate) {
			if dmaOn {
				fmt.Println("dma ended")
				dmaOn = false
			}
			soc.Cpu.StepCycle()
			cnt = 0
			continue
		}
	}

	for v := range 32 {
		sprite := soc.Ppu.OAM.NewSprite(v)
		fmt.Printf("%+v\n", sprite)
		//fmt.Println(sprite.GetVramFirstTileWordIndex())
	}
}
