package apu

type CPU struct {
	psram Memory

	r registers

	instructions       []Instruction
	currentInstruction Instruction

	Timers [3]*Timer

	resetSignal bool
	stopped     bool //apparently there is no way to wake up this cpu on the snes so stop and sleep are the same thing.
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

		//resetSignal: true,
	}
	ret.r.PC = 0xFFC0
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
	//cpu.r.PC = readResetVector()
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
