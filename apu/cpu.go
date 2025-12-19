package apu

import "SNES_emulator/memory"

type CPU struct {
	psram Memory

	r registers

	instructions       []Instruction
	currentInstruction Instruction

	Timers [3]*Timer

	stopped bool //apparently there is no way to wake up this cpu on the snes so stop and sleep are the same thing.
}

func NewCPU(psram Memory) *CPU {
	ret := &CPU{
		psram:              psram,
		instructions:       NewInstructionMap(),
		currentInstruction: nil,
		Timers: [3]*Timer{
			NewTimer(128),
			NewTimer(128),
			NewTimer(16),
		},
	}
	ret.Reset()

	return ret
}

// TODO create a separate APU struct that ticks timers/cpu/dsp individually
// that way timers dont have to be shared either
func NewApu(bus memory.Bus) *CPU {
	psram := NewSPCMemory()
	ret := NewCPU(psram)
	psram.Timers = &ret.Timers

	//probably the cleanest way
	bus.RegisterRange(0x2140, 0x217F, psram, "APU")
	return ret
}

func (cpu *CPU) StepCycle() bool {
	cpu.Timers[0].Tick()
	cpu.Timers[1].Tick()
	cpu.Timers[2].Tick()

	if cpu.stopped {
		return false
	}
	if cpu.currentInstruction == nil {
		cpu.currentInstruction = cpu.instructions[cpu.fetchByte()]
		cpu.currentInstruction.Reset()
		return false
	}
	if cpu.currentInstruction.Step(cpu) {
		cpu.currentInstruction = nil
		return true
	}

	return false
}

// TODO
func (cpu *CPU) Reset() {
	cpu.stopped = false
	cpu.r.PC = 0xFFC0
	cpu.r.PSW = 0
	//prolly reset the timers too??
}

func (cpu *CPU) fetchByte() byte {
	ret := cpu.psram.Read8(cpu.r.PC)
	cpu.r.PC++

	return ret
}

// PushByte pushes one byte onto the stack and updates SP.
func (cpu *CPU) PushByte(val byte) {
	cpu.psram.Write8(cpu.r.getStackAddr(), val)
	cpu.r.SP--
}

// PopByte pops one byte from the stack and updates SP.
func (cpu *CPU) PopByte() byte {
	cpu.r.SP++
	return cpu.psram.Read8(cpu.r.getStackAddr())
}
