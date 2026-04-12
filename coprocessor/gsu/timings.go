package gsu

import "SNES_emulator/internal/constants"

type accessTime struct {
	cart, cache uint64
}

var accessTimes = [2]accessTime{{cart: 6, cache: 2}, {cart: 5, cache: 1}}

type clock struct {
	cyclesTaken       uint64
	currentAccessTime accessTime

	r14Clock uint64

	r *registers
}

func (cl *clock) initClock(r *registers) {
	cl.r = r
	cl.currentAccessTime = accessTimes[0]
}

func (cl *clock) setAccessTime(clsr byte) {
	cl.currentAccessTime = accessTimes[clsr&1]
}

func (cl *clock) stepCache() {
	cl.cyclesTaken += cl.currentAccessTime.cache
}

func (cl *clock) stepCart() {
	cl.cyclesTaken += cl.currentAccessTime.cart
}

func (cl *clock) stepMultiplication(isFLMult bool) {
	isHighSpeed := cl.r.CFGR&CFGR_MS0 != 0
	if isFLMult {
		var baseCycle = uint64(7)
		if isHighSpeed {
			baseCycle = 3
		}
		cl.cyclesTaken += baseCycle << (cl.currentAccessTime.cache - 1)
	} else {
		if !isHighSpeed {
			cl.cyclesTaken += cl.currentAccessTime.cache
		}
	}
}

func (cl *clock) getSnesSideCycles() (cycles uint64) {
	cycles = cl.cyclesTaken >> constants.CYCLE_SHIFT
	cl.cyclesTaken -= cycles << constants.CYCLE_SHIFT
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
