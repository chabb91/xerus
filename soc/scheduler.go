package soc

import (
	"SNES_emulator/internal/constants"
	"time"
)

const DMA_OVERHEAD = constants.CYCLE_8
const CPU_REFRESH_DURATION = constants.CYCLE_40
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
	apuRatio := int64((float64(constants.SPU_BASE_FREQUENCY) /
		float64(soc.timing.baseFrequency)) *
		float64(PrecisionScale))

	var cyclesSinceLastInterval uint64
	cyclesPerPeriod := soc.timing.cyclesPerInterval
	soc.timing.start()

	for {
		cyclesSinceReset++
		cyclesSinceLastInterval++
		if cyclesSinceLastInterval == cyclesPerPeriod {
			cyclesSinceLastInterval = 0
			soc.timing.sync()
		}

		if ppuCnt > 1 {
			ppuCnt--
		} else {
			ppuCnt = soc.Ppu.Step()
		}

		apuDebt += apuRatio
		if apuDebt >= PrecisionScale { //only works because ratio << scale so 1 step is the max
			apuDebt -= PrecisionScale
			soc.Spu.Step()
		}

		if soc.Ppu.Refresh {
			//TODO there is some variation to this:
			//refresh pause begins at 538 cycles into the first scanline of the first frame,
			//and thereafter some multiple of 8 cycles after the previous pause that comes closest to 536
			soc.Ppu.Refresh = false
			cpuCnt += CPU_REFRESH_DURATION
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
	}
}

const INTERVAL_DIVIDER = 66
const BUSY_WAIT_TIME = time.Millisecond / 4

type timing struct {
	baseFrequency     uint64
	cyclesPerInterval uint64
	syncInterval      time.Duration

	expectedTime time.Time
}

func newTiming(isPal bool) *timing {
	var cycles, freq uint64
	if isPal {
		freq = constants.PAL_BASE_FREQUENCY
	} else {
		freq = constants.NTSC_BASE_FREQUENCY
	}
	cycles = freq / INTERVAL_DIVIDER
	syncInterval := time.Second / INTERVAL_DIVIDER
	return &timing{cyclesPerInterval: cycles, baseFrequency: freq, syncInterval: syncInterval}
}

func (t *timing) start() {
	t.expectedTime = time.Now()
}

func (t *timing) sync() {
	t.expectedTime = t.expectedTime.Add(t.syncInterval)

	diff := time.Until(t.expectedTime)

	//fmt.Println("Sleeping for : ", diff)
	time.Sleep(diff - BUSY_WAIT_TIME)
	for time.Now().Before(t.expectedTime) {
		//busy waiting for precision
	}
}
