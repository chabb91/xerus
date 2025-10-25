package soc

func (soc SoC) Run() {

	var cnt uint64
	var cnt2 uint64

	cpuTickRate := 3
	dmaTickRate := 4
	ppuTickRate := 2

	for {
		cnt++
		cnt2++
		soc.MulDiv.StepCycle()

		if cnt2 == uint64(ppuTickRate) {
			soc.Ppu.Step()
			cnt2 = 0
		}

		if soc.Dma.IsInProgress() && cnt == uint64(dmaTickRate) {
			soc.Dma.Step()
			cnt = 0
			continue

		}
		if !soc.Dma.IsInProgress() && cnt == uint64(cpuTickRate) {
			soc.Cpu.StepCycle()
			cnt = 0
			continue
		}

	}
}
