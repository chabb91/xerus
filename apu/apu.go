package apu

import (
	"SNES_emulator/apu/spc700"
	"SNES_emulator/memory"
)

type APU struct {
	psram *SPCMemory
	dsp   *DSP
	cpu   *spc700.CPU
}

func NewApu(bus memory.Bus) *APU {
	psram := NewSPCMemory()
	ret := &APU{
		psram: psram,
		dsp:   NewDsp(psram),
		cpu:   spc700.NewCPU(psram),
	}

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
