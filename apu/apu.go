package apu

import "SNES_emulator/memory"

type APU struct {
	psram *SPCMemory
	dsp   *DSP
	cpu   *CPU
}

func NewApu(bus memory.Bus) *APU {
	psram := NewSPCMemory()
	ret := &APU{}
	ret.psram = psram
	ret.dsp = NewDsp(psram)
	ret.cpu = NewCPU(psram)

	psram.dspRegs = ret.dsp

	//probably the cleanest way
	bus.RegisterRange(0x2140, 0x217F, psram, "APU")
	return ret
}

func (apu *APU) Step() {
	apu.cpu.StepCycle()
	apu.dsp.Step()
	apu.psram.TickTimers()
}
