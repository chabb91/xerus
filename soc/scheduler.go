package soc

const PPU_TICK_RATE = uint64(2)
const DMA_OVERHEAD = uint64(4)
const SPU_TICK_RATE = uint64(12)
const CPU_REFRESH_DURATION = uint64(20)
const CPU_DELAY_INTERRUPT_AFTER_DMA = uint64(2) //cpu cycle count

func (soc SoC) Run() {
	var cnt uint64 //the dma/cpu cycle counter is counting down due to the variable access speed
	var cnt1 uint64
	var cnt2 uint64
	var cyclesSinceReset uint64
	var cyclesSincePause uint64
	var nmiDelay uint64

	var prevDmaActive bool
	var nmiSignalBeforeDma bool
	var irqSignalBeforeDma bool
	var nmiTriggeredDuringDma bool
	var irqTriggeredDuringDma bool

	for {
		cnt1++
		cnt2++
		cyclesSinceReset++

		if cnt1 == SPU_TICK_RATE {
			soc.Spu.Step()
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
			soc.MulDiv.StepCycle()
			soc.Cpu.StepCycle()
			cnt = soc.Cpu.CyclesTaken

			if nmiTriggeredDuringDma || irqTriggeredDuringDma {
				if nmiDelay > 1 {
					nmiDelay--
				} else {
					soc.Cpu.NmiSignal = nmiTriggeredDuringDma
					soc.Cpu.IrqSignal = irqTriggeredDuringDma
					nmiTriggeredDuringDma = false
					irqTriggeredDuringDma = false
				}
			}

			if dmaHandoff {
				cyclesSincePause = cyclesSinceReset + cnt
				alignment := (4 - ((cyclesSincePause) & 3)) // TODO this should be sinceReset+nextCycleCnt
				cnt += DMA_OVERHEAD + alignment
				nmiSignalBeforeDma = soc.Cpu.NmiSignal
				irqSignalBeforeDma = soc.Cpu.IrqSignal
			}
			prevDmaActive = dmaActive
		} else {
			if soc.Dma.Mdmaen != 0 {
				if !nmiSignalBeforeDma && soc.Cpu.NmiSignal {
					nmiDelay = CPU_DELAY_INTERRUPT_AFTER_DMA
					soc.Cpu.NmiSignal = false
					nmiTriggeredDuringDma = true
				}
				if !irqSignalBeforeDma && soc.Cpu.IrqSignal {
					nmiDelay = CPU_DELAY_INTERRUPT_AFTER_DMA
					soc.Cpu.IrqSignal = false
					irqTriggeredDuringDma = true
				}
			}

			cnt = soc.Dma.Step()

			prevDmaActive = soc.Dma.IsInProgress()
			if !prevDmaActive {
				alignment := (4 - (((cyclesSinceReset + cnt) - cyclesSincePause) & 3))
				cnt += alignment
			}
		}
	}
}
