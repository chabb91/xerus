package spc700

type Memory interface {
	Read8(addr uint16) byte
	Write8(addr uint16, val byte)
}

type CPU struct {
	psram Memory

	r registers

	instructions       []Instruction
	currentInstruction Instruction

	stopped bool //apparently there is no way to wake up this cpu on the snes so stop and sleep are the same thing.
}

func NewCPU(psram Memory) *CPU {
	ret := &CPU{
		psram:              psram,
		instructions:       NewInstructionMap(),
		currentInstruction: nil,
	}
	ret.Reset()

	return ret
}

func (cpu *CPU) StepCycle() bool {
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
