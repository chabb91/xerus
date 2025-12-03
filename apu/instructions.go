package apu

// Instruction represents a single CPU instruction, executed one cycle at a time.
type Instruction interface {
	// Step performs one cycle of the instruction's execution.
	// It returns true if the instruction is complete, false otherwise.
	Step(cpu *CPU) bool
	Reset(cpu *CPU)
}

func NewInstructionMap() []Instruction {
	ret := make([]Instruction, 0x100)

	ret[0x5F] = &JmpAbs{}
	return ret
}

type JmpAbs struct {
	state int
	lo    byte
}

func (i *JmpAbs) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lo = cpu.fetchByte()
		i.state++
	case 1:
		hi := cpu.fetchByte()
		cpu.r.PC = uint16(hi)<<8 | uint16(i.lo)
		return true
	}
	return false
}

func (i *JmpAbs) Reset(cpu *CPU) {
	i.state = 0
}
