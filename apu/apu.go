package apu

import (
	"github.com/chabb91/xerus/apu/spc700"
	"github.com/chabb91/xerus/memory"
)

type APU struct {
	psram *SPCMemory
	Dsp   *DSP
	cpu   *spc700.CPU
}

func NewApu(bus memory.Bus) *APU {
	psram := NewSPCMemory()
	ret := &APU{
		psram: psram,
		Dsp:   NewDsp(psram),
		cpu:   spc700.NewCPU(psram),
	}

	psram.dspRegs = ret.Dsp

	//probably the cleanest way
	bus.RegisterRange(0x2140, 0x217F, psram, "APU")
	return ret
}

func (apu *APU) Step() {
	apu.cpu.StepCycle()
	apu.Dsp.Step()
	apu.psram.TickTimers()
}
