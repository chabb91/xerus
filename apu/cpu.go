package apu

const PSRAM_SIZE = 0x10000

type CPU struct {
	psram [PSRAM_SIZE]byte

	r registers

	instructions       []Instruction
	currentInstruction Instruction

	resetSignal bool
}

func NewCPU() *CPU {
	return &CPU{
		instructions:       NewInstructionMap(),
		currentInstruction: nil,

		resetSignal: true,
	}
}

func (cpu *CPU) StepCycle() bool {
	if cpu.currentInstruction == nil {
		cpu.currentInstruction = cpu.instructions[cpu.fetchByte()]
		cpu.currentInstruction.Reset(cpu)
		return false
	}
	if cpu.currentInstruction.Step(cpu) {
		cpu.currentInstruction = nil
		return true
	}
	return false
}

func (cpu *CPU) fetchByte() byte {
	ret := cpu.psram[cpu.r.PC]
	cpu.r.PC++

	return ret
}

// PushByte pushes one byte onto the stack and updates SP.
func (cpu *CPU) PushByte(val byte) {
	cpu.psram[cpu.r.getStackAddr()] = val
	cpu.r.SP--
}

// PopByte pops one byte from the stack and updates SP.
func (cpu *CPU) PopByte() byte {
	cpu.r.SP++
	return cpu.psram[cpu.r.getStackAddr()]
}
