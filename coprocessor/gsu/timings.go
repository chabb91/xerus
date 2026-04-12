package gsu

import (
	"SNES_emulator/internal/constants"
)

type accessTime struct {
	cart, cache uint64
}

var accessTimes = [2]accessTime{{cart: 6, cache: 2}, {cart: 5, cache: 1}}

func (gsu *GSU) setAccessTime(clsr byte) {
	gsu.currentAccessTime = accessTimes[clsr&1]
}

func (gsu *GSU) stepCache() {
	gsu.cyclesTaken += gsu.currentAccessTime.cache
}

func (gsu *GSU) stepCart() {
	gsu.cyclesTaken += gsu.currentAccessTime.cart
}

func (gsu *GSU) stepMultiplication(isFLMult bool) {
	isHighSpeed := gsu.r.CFGR&CFGR_MS0 != 0
	if isFLMult {
		var baseCycle = uint64(7)
		if isHighSpeed {
			baseCycle = 3
		}
		gsu.cyclesTaken += baseCycle << (gsu.currentAccessTime.cache - 1)
	} else {
		if !isHighSpeed {
			gsu.cyclesTaken += gsu.currentAccessTime.cache
		}
	}
}

func (gsu *GSU) getSnesSideCycles() (cycles uint64) {
	cycles = gsu.cyclesTaken >> constants.CYCLE_SHIFT
	gsu.cyclesTaken -= cycles << constants.CYCLE_SHIFT
	return
}

func (gsu *GSU) stepRomAddrPtr() {
	if gsu.r.r14Modified {
		gsu.r14Clock = gsu.currentAccessTime.cart
		gsu.r.r14Modified = false
		gsu.r.setFlag(FlagR, true)
	} else {
		if gsu.r14Clock != 0 {
			gsu.r14Clock -= min(gsu.r14Clock, gsu.cyclesTaken)
			if gsu.r14Clock == 0 {
				gsu.r.setFlag(FlagR, false)
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
		gsu.r.setFlag(FlagR, false)
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

func (gsu *GSU) stepRamWriteCache() {
	if gsu.ramWriteCacheClock != 0 {
		gsu.ramWriteCacheClock -= min(gsu.ramWriteCacheClock, gsu.cyclesTaken)
	}
}

// tracks if an instruction accessed rom/ram when RON/RAN was disabled.
// this causes the cpu to WAIT till it is re-enabled.
type waitState struct {
	waitForRom, waitForRam bool
	waiting                bool
}

func (w *waitState) updateWait(scmr byte) {
	if w.waitForRam {
		w.waitForRam = scmr&RAN == 0
	}
	if w.waitForRom {
		w.waitForRom = scmr&RON == 0
	}
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRomOwnership(scmr byte) {
	w.waitForRom = scmr&RON == 0
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRamOwnership(scmr byte) {
	w.waitForRam = scmr&RAN == 0
	w.waiting = w.waitForRam || w.waitForRom
}
