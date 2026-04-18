package gsu

import (
	"SNES_emulator/internal/constants"
)

type accessTime struct {
	cart, cache uint64
}

// the Bsnes values are commented out. there is something I dont understand about timings
// so the supposedly correct values end up being way too slow
// all timings are represented as 21mhz, so 1 cycle = 1/21_000_000

// var accessTimes = [2]accessTime{{cart: 6, cache: 2}, {cart: 5, cache: 1}}
var accessTimes = [2]accessTime{{cart: 4, cache: 2}, {cart: 3, cache: 1}}

func (gsu *GSU) setAccessTime(clsr byte) {
	gsu.currentAccessTime = accessTimes[clsr&1]
}

func (gsu *GSU) stepCache() {
	step := gsu.currentAccessTime.cache
	gsu.stepRomAddrPtr(step)
	gsu.stepRamWriteCache(step)
	gsu.cyclesTaken += step
}

func (gsu *GSU) stepCart() {
	step := gsu.currentAccessTime.cart
	gsu.stepRomAddrPtr(step)
	gsu.stepRamWriteCache(step)
	gsu.cyclesTaken += step
}

func (gsu *GSU) stepMultiplication(isFLMult bool) {
	step := uint64(0)
	isHighSpeed := hasFlag(gsu.r.CFGR, MS0)
	if isFLMult {
		baseCycle := uint64(7)
		if isHighSpeed {
			baseCycle = 3
		}
		step = baseCycle << (gsu.currentAccessTime.cache - 1)
	} else {
		if !isHighSpeed {
			step = gsu.currentAccessTime.cache
		}
	}
	gsu.stepRomAddrPtr(step)
	gsu.stepRamWriteCache(step)
	gsu.cyclesTaken += step
}

func (gsu *GSU) getSnesSideCycles() (cycles uint64) {
	cycles = gsu.cyclesTaken >> constants.CYCLE_SHIFT
	gsu.cyclesTaken -= cycles << constants.CYCLE_SHIFT
	return
}

func (gsu *GSU) stepRomAddrPtr(step uint64) {
	if gsu.r.r14Modified {
		gsu.r14Clock = gsu.currentAccessTime.cart
		gsu.r.r14Modified = false
		setFlag(&gsu.r.SFR, FlagR, true)
	} else {
		if gsu.r14Clock != 0 {
			gsu.r14Clock -= min(gsu.r14Clock, step)
			if gsu.r14Clock == 0 {
				setFlag(&gsu.r.SFR, FlagR, false)
				val, _ := gsu.Read8(gsu.r.ROMBR, gsu.r.cpuRegisters[14])
				gsu.r.romAddrPtr = val
			}
		}
	}
}

func (gsu *GSU) readRomAddrPtr() byte {
	if gsu.r14Clock != 0 {
		val, _ := gsu.Read8(gsu.r.ROMBR, gsu.r.cpuRegisters[14])
		gsu.cyclesTaken += gsu.r14Clock
		gsu.r.romAddrPtr = val
		setFlag(&gsu.r.SFR, FlagR, false)
		gsu.r14Clock = 0
	}
	return gsu.r.romAddrPtr
}

func (gsu *GSU) waitRamWriteCacheFlush() {
	if gsu.ramWriteCacheClock != 0 {
		gsu.cyclesTaken += gsu.ramWriteCacheClock
		gsu.ramWriteCacheClock = 0
	}
}

func (gsu *GSU) incrementRamWriteCacheClock() {
	gsu.ramWriteCacheClock += gsu.currentAccessTime.cart
}

func (gsu *GSU) stepRamWriteCache(step uint64) {
	if gsu.ramWriteCacheClock != 0 {
		gsu.ramWriteCacheClock -= min(gsu.ramWriteCacheClock, step)
	}
}

// tracks if an instruction accessed rom/ram when RON/RAN was disabled.
// this causes the cpu to WAIT till it is re-enabled.
type waitState struct {
	waitForRom, waitForRam bool
	waiting                bool
}

func (w *waitState) updateWait(scmr scmr) {
	if w.waitForRam {
		w.waitForRam = scmr&RAN == 0
	}
	if w.waitForRom {
		w.waitForRom = scmr&RON == 0
	}
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRomOwnership(scmr scmr) {
	w.waitForRom = scmr&RON == 0
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRamOwnership(scmr scmr) {
	w.waitForRam = scmr&RAN == 0
	w.waiting = w.waitForRam || w.waitForRom
}
