package soc

import (
	"time"
)

const PPU_TICK_RATE = uint64(2)
const DMA_OVERHEAD = uint64(4)
const CPU_REFRESH_DURATION = uint64(20)
const CPU_DELAY_INTERRUPT_AFTER_DMA = uint64(2) //cpu cycle count

func (soc SoC) Run() {
	var cpuCnt uint64 //the dma/cpu cycle counter is counting down due to the variable access speed
	var ppuCnt uint64
	var cyclesSinceReset uint64
	var cyclesSincePause uint64
	var nmiDelay uint64

	var prevDmaActive bool
	var nmiSignalBeforeDma bool
	var irqSignalBeforeDma bool
	var nmiTriggeredDuringDma bool
	var irqTriggeredDuringDma bool

	var apuDebt int64
	PrecisionScale := int64(1_000_000_000)
	apuRatio := int64((float64(SPU_BASE_FREQUENCY) /
		float64(soc.timing.cyclesPerPeriod*INTERVAL_DIVIDER)) *
		float64(PrecisionScale))

	var cyclesSinceLastInterval uint64
	cyclesPerPeriod := soc.timing.cyclesPerPeriod
	soc.timing.start()

	for {
		ppuCnt++
		cyclesSinceReset++
		cyclesSinceLastInterval++
		if cyclesSinceLastInterval == cyclesPerPeriod {
			cyclesSinceLastInterval = 0
			soc.timing.sync()
		}

		if ppuCnt == PPU_TICK_RATE {
			soc.Ppu.Step()
			ppuCnt = 0
		}

		if cpuCnt > 1 {
			cpuCnt--
			continue
		}

		dmaActive := soc.Dma.IsInProgress()
		dmaHandoff := dmaActive && !prevDmaActive
		if !dmaActive || dmaHandoff {
			soc.MulDiv.StepCycle()
			soc.Cpu.StepCycle()
			cpuCnt = soc.Cpu.CyclesTaken

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
				cyclesSincePause = cyclesSinceReset + cpuCnt
				alignment := (4 - ((cyclesSincePause) & 3)) // TODO this should be sinceReset+nextCycleCnt
				cpuCnt += DMA_OVERHEAD + alignment
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

			cpuCnt = soc.Dma.Step()

			prevDmaActive = soc.Dma.IsInProgress()
			if !prevDmaActive {
				alignment := (4 - (((cyclesSinceReset + cpuCnt) - cyclesSincePause) & 3))
				cpuCnt += alignment
			}
		}

		if soc.Ppu.Refresh {
			//TODO there is some variation to this:
			//refresh pause begins at 538 cycles into the first scanline of the first frame,
			//and thereafter some multiple of 8 cycles after the previous pause that comes closest to 536
			soc.Ppu.Refresh = false
			cpuCnt += CPU_REFRESH_DURATION
		}

		apuDebt += int64(cpuCnt) * apuRatio
		for apuDebt >= PrecisionScale {
			apuDebt -= PrecisionScale
			soc.Spu.Step()
		}
	}
}

const INTERVAL_DIVIDER = 66
const PAL_BASE_FREQUENCY = 21_281_370
const NTSC_BASE_FREQUENCY = 1_890_000_000 / 88
const PAL_CYCLES_PER_INTERVAL = uint64(PAL_BASE_FREQUENCY/INTERVAL_DIVIDER) / 2
const NTSC_CYCLES_PER_INTERVAL = uint64(NTSC_BASE_FREQUENCY/INTERVAL_DIVIDER) / 2

const CLOCK_SYNC_INTERVAL = time.Second / INTERVAL_DIVIDER
const BUSY_WAIT_TIME = time.Millisecond / 4

const SPU_BASE_FREQUENCY = 1_024_000

type timing struct {
	cyclesPerPeriod uint64

	expectedTime time.Time
}

func newTiming(isPal bool) *timing {
	var cycles uint64
	if isPal {
		cycles = PAL_CYCLES_PER_INTERVAL
	} else {
		cycles = NTSC_CYCLES_PER_INTERVAL
	}
	return &timing{cyclesPerPeriod: cycles}
}

func (t *timing) start() {
	t.expectedTime = time.Now()
}

func (t *timing) sync() {
	t.expectedTime = t.expectedTime.Add(CLOCK_SYNC_INTERVAL)

	diff := time.Until(t.expectedTime)

	//fmt.Println("Sleeping for : ", diff)
	time.Sleep(diff - BUSY_WAIT_TIME)
	for time.Now().Before(t.expectedTime) {
		//busy waiting for precision
	}
}
