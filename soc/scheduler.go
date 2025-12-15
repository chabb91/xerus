package soc

const PPU_TICK_RATE = uint64(2)
const DMA_OVERHEAD = uint64(4)
const CPU_BASE_TICK_RATE = uint64(3)
const SPU_TICK_RATE = uint64(12)
const CPU_REFRESH_DURATION = uint64(20)

func (soc SoC) Run() {
	var cnt uint64 //the dma/cpu cycle counter is counting down due to the variable access speed
	var cnt1 uint64
	var cnt2 uint64
	var cyclesSinceReset uint64
	var cyclesSincePause uint64

	var prevDmaActive bool

	for {
		cnt1++
		cnt2++
		cyclesSinceReset++

		soc.MulDiv.StepCycle()
		if cnt1 == SPU_TICK_RATE {
			soc.Spu.StepCycle()
			cnt1 = 0
		}
		if cnt2 == PPU_TICK_RATE {
			soc.Ppu.Step()
			cnt2 = 0
		}

		if soc.Ppu.Refresh {
			//TODO there is some variation to this:
			//refresh pause begins at 538 cycles into the first scanline of the first frame,
			//and thereafter some multiple of 8 cycles after the previous pause that comes closest to 536
			soc.Ppu.Refresh = false
			cnt += CPU_REFRESH_DURATION
		}

		if cnt > 1 {
			cnt--
			continue
		}

		dmaActive := soc.Dma.IsInProgress()
		dmaHandoff := dmaActive && !prevDmaActive
		if !dmaActive || dmaHandoff {
			soc.Cpu.StepCycle()
			cnt = CPU_BASE_TICK_RATE //TODO introduce variable cycle count

			if dmaHandoff {
				cyclesSincePause = cyclesSinceReset + cnt
				alignment := (4 - ((cyclesSincePause) & 3)) // TODO this should be sinceReset+nextCycleCnt
				cnt += DMA_OVERHEAD + alignment
			}
			prevDmaActive = dmaActive
		} else {
			cnt = soc.Dma.Step()

			dmaStillActive := soc.Dma.IsInProgress()
			if !dmaStillActive {
				alignment := (4 - (((cyclesSinceReset + cnt) - cyclesSincePause) & 3))
				cnt += alignment
			}
			prevDmaActive = dmaStillActive
		}
	}
}
