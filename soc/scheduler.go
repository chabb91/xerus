package soc

func (soc SoC) Run() {

	var cnt uint64
	var cnt2 uint64

	cpuTickRate := 3
	dmaTickRate := 4
	ppuTickRate := 2

	var dmaOn bool
	for {
		cnt++
		cnt2++
		soc.MulDiv.StepCycle()

		if cnt2 == uint64(ppuTickRate) {
			soc.Ppu.Step()
			cnt2 = 0
		}

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
}
