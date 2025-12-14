package soc

func (soc SoC) Run() {

	var cnt uint64
	var cnt1 uint64
	var cnt2 uint64
	var cyclesSinceReset uint64

	var prevDmaActive bool

	cpuTickRate := 3
	cpuRefreshDuration := 20
	//dmaTickRate := 4
	ppuTickRate := 2
	spuTickRate := 12

	cnt1 = uint64(spuTickRate) - 1
	cnt2 = uint64(ppuTickRate) - 1

	for {
		cnt1++
		cnt2++
		cyclesSinceReset++

		soc.MulDiv.StepCycle()
		if cnt1 == uint64(spuTickRate) {
			soc.Spu.StepCycle()
			cnt1 = 0
		}
		if cnt2 == uint64(ppuTickRate) {
			soc.Ppu.Step()
			cnt2 = 0
		}

		if soc.Ppu.Refresh {
			//TODO there is some variation to this:
			//refresh pause begins at 538 cycles into the first scanline of the first frame,
			//and thereafter some multiple of 8 cycles after the previous pause that comes closest to 536
			soc.Ppu.Refresh = false
			cnt += uint64(cpuRefreshDuration) - 1
		}

		if cnt > 0 {
			cnt--
			continue
		}

		// TODO if i keep decrementing like this in the future the cycle counts should be adjusted instead
		// so theres no unnecessary subtraction
		dmaActive := soc.Dma.IsInProgress()
		if !dmaActive || (dmaActive && !prevDmaActive) {
			soc.Cpu.StepCycle()

			if dmaActive && !prevDmaActive {
				alignment := (4 - (cyclesSinceReset & 3)) // TODO this should be sinceReset+nextCycleCnt
				cnt = 4 + alignment - 1
			} else {
				cnt = uint64(cpuTickRate) - 1
			}

			prevDmaActive = dmaActive
		} else {
			cycles := soc.Dma.Step()
			dmaStillActive := soc.Dma.IsInProgress()
			if !dmaStillActive {
				alignment := (4 - (cyclesSinceReset & 3)) //FIXME this should be cycles after pause not after reset
				cnt = cycles + alignment - 1
			} else {
				cnt = cycles - 1
			}

			prevDmaActive = dmaStillActive
		}
	}
}
