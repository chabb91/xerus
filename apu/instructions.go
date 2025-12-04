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

	//BBC
	ret[0x13] = &DirectPageBitRelative{branchOnZero: true, mask: 0x01}
	ret[0x33] = &DirectPageBitRelative{branchOnZero: true, mask: 0x02}
	ret[0x53] = &DirectPageBitRelative{branchOnZero: true, mask: 0x04}
	ret[0x73] = &DirectPageBitRelative{branchOnZero: true, mask: 0x08}
	ret[0x93] = &DirectPageBitRelative{branchOnZero: true, mask: 0x10}
	ret[0xB3] = &DirectPageBitRelative{branchOnZero: true, mask: 0x20}
	ret[0xD3] = &DirectPageBitRelative{branchOnZero: true, mask: 0x40}
	ret[0xF3] = &DirectPageBitRelative{branchOnZero: true, mask: 0x80}
	//BBS
	ret[0x03] = &DirectPageBitRelative{branchOnZero: false, mask: 0x01}
	ret[0x23] = &DirectPageBitRelative{branchOnZero: false, mask: 0x02}
	ret[0x43] = &DirectPageBitRelative{branchOnZero: false, mask: 0x04}
	ret[0x63] = &DirectPageBitRelative{branchOnZero: false, mask: 0x08}
	ret[0x83] = &DirectPageBitRelative{branchOnZero: false, mask: 0x10}
	ret[0xA3] = &DirectPageBitRelative{branchOnZero: false, mask: 0x20}
	ret[0xC3] = &DirectPageBitRelative{branchOnZero: false, mask: 0x40}
	ret[0xE3] = &DirectPageBitRelative{branchOnZero: false, mask: 0x80}
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

// used for the 8 BBC and 8 BBS instructions
type DirectPageBitRelative struct {
	state        int
	lo           byte
	shouldBranch bool

	mask         byte
	branchOnZero bool
}

func (i *DirectPageBitRelative) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lo = cpu.fetchByte()
		i.state++
	case 1:
		i.lo = cpu.psram.Read8(uint16(cpu.r.getDirectPageNum())<<8 | uint16(i.lo))
		i.state++
	case 2:
		result := i.lo & i.mask
		i.shouldBranch = (result == 0) == i.branchOnZero
		i.state++
	case 3:
		i.lo = cpu.fetchByte()
		if i.shouldBranch {
			cpu.r.PC += uint16(int8(i.lo))
			i.state++
		} else {
			return true
		}
	case 4:
		i.state++
	case 5:
		i.state++
		return true
	}
	return false
}

func (i *DirectPageBitRelative) Reset(cpu *CPU) {
	i.state = 0
}
