package main

import (
	"SNES_emulator/ppu"
	"SNES_emulator/soc"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"
)

func main() {
	soc := soc.NewSoC()
	var cnt uint64

	cpuTickRate := 3
	dmaTickRate := 4

	start := time.Now()
	var dmaOn bool
	for range 27000 {
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
	//soc.Ppu.VRAM.VRAM[0x7C00] = 35
	//soc.Ppu.VRAM.VRAM[0x7C01] = 79
	//soc.Ppu.VRAM.VRAM[0x7C02] = 111
	soc.Ppu.VRAM.VRAM[0x7C00] = 72
	soc.Ppu.VRAM.VRAM[0x7C01] = 69
	soc.Ppu.VRAM.VRAM[0x7C02] = 76
	soc.Ppu.VRAM.VRAM[0x7C03] = 76
	soc.Ppu.VRAM.VRAM[0x7C04] = 79
	soc.Ppu.VRAM.VRAM[0x7C05] = 32
	soc.Ppu.VRAM.VRAM[0x7C06] = 77
	soc.Ppu.VRAM.VRAM[0x7C07] = 97
	soc.Ppu.VRAM.VRAM[0x7C08] = 114
	soc.Ppu.VRAM.VRAM[0x7C09] = 32
	soc.Ppu.VRAM.VRAM[0x7C0A] = 102
	soc.Ppu.VRAM.VRAM[0x7C0B] = 114
	soc.Ppu.VRAM.VRAM[0x7C0C] = 111
	soc.Ppu.VRAM.VRAM[0x7C0D] = 109
	soc.Ppu.VRAM.VRAM[0x7C0E] = 32
	soc.Ppu.VRAM.VRAM[0x7C0F] = 83
	soc.Ppu.VRAM.VRAM[0x7C10] = 78
	soc.Ppu.VRAM.VRAM[0x7C11] = 69
	soc.Ppu.VRAM.VRAM[0x7C12] = 83
	soc.Ppu.VRAM.VRAM[0x7C13] = 33
	//fmt.Println(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, 2, 5))
	//for v := range 32 {
	//	sprite := soc.Ppu.OAM.NewSprite(v)
	//	fmt.Printf("%+v\n", sprite)
	//	//fmt.Println(sprite.GetVramFirstTileWordIndex())
	//}

	//for v := range 8 {
	//fmt.Println(soc.Ppu.VRAM.VRAM[0x7C00+v])
	//fmt.Println(soc.Ppu.VRAM.VRAM[512+v])
	//}
	img := image.NewNRGBA(image.Rect(0, 0, 256, 240))
	for H := range 256 {
		for V := range 240 {
			img.Set(H, V, ppu.SNESColorToARGB(soc.Ppu.Bg1.GetDotAt(soc.Ppu.VRAM.VRAM, soc.Ppu.CGRAM.CGRAM, byte(H), byte(V))))
		}
	}
	fmt.Printf("Execution time: %v\n", time.Since(start))

	outFile, err := os.Create("helloworld.png")
	if err != nil {
		return
	}
	defer outFile.Close()

	// 4. Encode the image data to the file using the PNG encoder.
	if err := png.Encode(outFile, img); err != nil {
		return
	}
}
