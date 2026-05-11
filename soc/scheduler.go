package soc

import (
	"time"

	"github.com/chabb91/xerus/internal/constants"
)

const CPU_DELAY_INTERRUPT_AFTER_DMA = uint64(2) //cpu cycle count

func (soc SoC) Run() {
	var cpuCnt uint64
	var ppuCnt uint64
	var copCnt uint64
	var prevApuStep uint64
	var cyclesAtPause uint64
	var nmiDelay uint64

	var prevDmaActive bool = false
	var nmiSignalBeforeDma bool
	var irqSignalBeforeDma bool
	var nmiTriggeredDuringDma bool
	var irqTriggeredDuringDma bool

	var apuDebt uint64
	PrecisionScale := uint64(1_000_000_000)
	apuRatio := uint64((float64(constants.SPU_BASE_FREQUENCY) /
		float64(soc.timing.baseFrequency)) *
		float64(PrecisionScale))

	var cyclesSinceLastInterval uint64
	cyclesPerPeriod := soc.timing.cyclesPerInterval
	soc.timing.start()

	stepPpu := func() {
		ppuCnt += soc.Ppu.Step()
		if soc.Ppu.Refresh {
			//TODO there is some variation to this:
			//refresh pause begins at 538 cycles into the first scanline of the first frame,
			//and thereafter some multiple of 8 cycles after the previous pause that comes closest to 536
			soc.Ppu.Refresh = false
			cpuCnt += constants.CYCLE_40
		}
	}
	var stepCop func()
	var getNextStep func() uint64
	if soc.Cop == nil {
		stepCop = func() { return }
		getNextStep = func() uint64 { return min(ppuCnt, cpuCnt) }
	} else {
		stepCop = func() { copCnt += soc.Cop.Step() }
		getNextStep = func() uint64 { return min(ppuCnt, copCnt, cpuCnt) }
	}

	syncApu := func(nextStep uint64) {
		apuDebt += apuRatio * (nextStep - prevApuStep)
		for apuDebt >= PrecisionScale {
			apuDebt -= PrecisionScale
			soc.Spu.Step()
		}
		prevApuStep = nextStep
	}

	stepCpu := func() {
		if !prevDmaActive {
			prevDmaActive = soc.Dma.IsInProgress()
			soc.MulDiv.StepCycle()
			soc.Cpu.StepCycle()
			cpuCnt += soc.Cpu.CyclesTaken

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

			if prevDmaActive { //handoff
				cyclesAtPause = cpuCnt
				// TODO this should be sinceReset+nextCycleCnt
				cpuCnt += (constants.CYCLE_8 - ((cyclesAtPause) &
					(constants.CYCLE_8 - 1)))
				cpuCnt += constants.CYCLE_8 //transfer overhead
				nmiSignalBeforeDma = soc.Cpu.NmiSignal
				irqSignalBeforeDma = soc.Cpu.IrqSignal
			}
		} else {
			//if soc.Dma.Mdmaen != 0 {
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
			//}

			cpuCnt += soc.Dma.Step()

			prevDmaActive = soc.Dma.IsInProgress()
			if !prevDmaActive {
				cpuCnt += (constants.CYCLE_8 - (((cpuCnt) - cyclesAtPause) &
					(constants.CYCLE_8 - 1)))
			}
		}
	}

	for {
		nextStep := getNextStep()
		if nextStep-cyclesSinceLastInterval >= cyclesPerPeriod {
			cyclesSinceLastInterval = nextStep
			soc.timing.sync()
		}

		if ppuCnt == nextStep {
			stepPpu()
		}

		if copCnt == nextStep {
			stepCop()
		}

		if cpuCnt == nextStep {
			syncApu(nextStep)
			stepCpu()
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
