package soc

func (soc SoC) Run() {

	var cnt uint64
	var cnt1 uint64
	var cnt2 uint64

	cpuTickRate := 3
	//dmaTickRate := 4
	ppuTickRate := 2
	spuTickRate := 12

	cnt1 = uint64(spuTickRate) - 1
	cnt2 = uint64(ppuTickRate) - 1

	for {
		cnt1++
		cnt2++
		soc.MulDiv.StepCycle()
		if cnt1 == uint64(spuTickRate) {
			soc.Spu.StepCycle()
			cnt1 = 0
		}
		if cnt2 == uint64(ppuTickRate) {
			soc.Ppu.Step()
			cnt2 = 0
		}

		if cnt > 0 {
			cnt--
			continue
		}

		//TODO if i keep decrementing like this in the future the cycle counts should be adjusted instead
		//so theres no unnecessary subtraction
		if soc.Dma.IsInProgress() {
			cycles := soc.Dma.Step()
			cnt = cycles - 1
		} else {
			soc.Cpu.StepCycle()
			cnt = uint64(cpuTickRate) - 1
		}
	}
}
