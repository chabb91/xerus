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
	for range 55000 {
		cnt++
		soc.MulDiv.StepCycle()
		if soc.Dma.Mdmaen != 0 && cnt == uint64(dmaTickRate) {
			if !dmaOn {
				dmaOn = true
			}
			soc.Dma.Step()
			cnt = 0
			continue

		}
		if soc.Dma.Mdmaen == 0 && cnt == uint64(cpuTickRate) {
			if dmaOn {
				dmaOn = false
			}
			soc.Cpu.StepCycle()
			cnt = 0
			continue
		}
	}
	soc.Ppu.VRAM.VRAM[0x7C00] = 35
	fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 2, 5))
	fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 2, 5))
	fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 7, 5))
	fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 7, 6))
	fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 8, 6))
	//for v := range 32 {
	//	sprite := soc.Ppu.OAM.NewSprite(v)
	//	fmt.Printf("%+v\n", sprite)
	//	//fmt.Println(sprite.GetVramFirstTileWordIndex())
	//}

	//for v := range 8 {
	//fmt.Println(soc.Ppu.VRAM.VRAM[0x7C00+v])
	//fmt.Println(soc.Ppu.VRAM.VRAM[512+v])
	//}

}
