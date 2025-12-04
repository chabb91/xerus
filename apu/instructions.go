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

	ret[0x5F] = &JmpAbs{indirect: false}
	ret[0x1F] = &JmpAbs{indirect: true}

	//BBC
	branchIfEqual := func(_ *CPU, b byte, _ uint16) bool { return b == 0 }
	ret[0x13] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x01}}
	ret[0x33] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x02}}
	ret[0x53] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x04}}
	ret[0x73] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x08}}
	ret[0x93] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x10}}
	ret[0xB3] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x20}}
	ret[0xD3] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x40}}
	ret[0xF3] = &Relative{branchCondition: branchIfEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x80}}
	//BBS
	branchIfNotEqual := func(_ *CPU, b byte, _ uint16) bool { return b != 0 }
	ret[0x03] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x01}}
	ret[0x23] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x02}}
	ret[0x43] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x04}}
	ret[0x63] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x08}}
	ret[0x83] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x10}}
	ret[0xA3] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x20}}
	ret[0xC3] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x40}}
	ret[0xE3] = &Relative{branchCondition: branchIfNotEqual, am: &DirectPage{io: READ_RAM, mode: BIT, mask: 0x80}}
	//DBNZ
	ret[0x6E] = &Relative{branchCondition: func(c *CPU, b byte, a uint16) bool { b--; c.psram.Write8(a, b); return b != 0 },
		am: &DirectPage{mode: DEFAULT, io: READ_RAM}}
	ret[0xFE] = &Relative{branchCondition: func(c *CPU, b byte, a uint16) bool { c.r.Y--; return c.r.Y != 0 },
		am: &AccessRegister{mode: REGISTER_Y}}
	//CBNE
	branchIfAccumulatorNotEqual := func(c *CPU, b byte, _ uint16) bool { return b != c.r.A }
	ret[0xDE] = &Relative{branchCondition: branchIfAccumulatorNotEqual, am: &DirectPage{mode: X_INDEXED, io: READ_RAM}}
	ret[0x2E] = &Relative{branchCondition: branchIfAccumulatorNotEqual, am: &DirectPage{mode: DEFAULT, io: READ_RAM}}
	//standard branch operations: BRA, BEQ, BNE, BCS, BCC, BVS, BVC, BMI, BPL
	ret[0x2F] = &Relative{branchCondition: func(_ *CPU, _ byte, _ uint16) bool { return true }, am: nil}
	ret[0xF0] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return c.r.hasFlag(FlagZ) }, am: nil}
	ret[0xD0] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return !c.r.hasFlag(FlagZ) }, am: nil}
	ret[0xB0] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return c.r.hasFlag(FlagC) }, am: nil}
	ret[0x90] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return !c.r.hasFlag(FlagC) }, am: nil}
	ret[0x70] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return c.r.hasFlag(FlagV) }, am: nil}
	ret[0x50] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return !c.r.hasFlag(FlagV) }, am: nil}
	ret[0x30] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return c.r.hasFlag(FlagN) }, am: nil}
	ret[0x10] = &Relative{branchCondition: func(c *CPU, _ byte, _ uint16) bool { return !c.r.hasFlag(FlagN) }, am: nil}

	return ret
}

type JmpAbs struct {
	state  int
	lo, hi byte
	addr   uint16

	indirect bool
}

func (i *JmpAbs) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lo = cpu.fetchByte()
		i.state++
	case 1:
		i.hi = cpu.fetchByte()
		i.addr = uint16(i.hi)<<8 | uint16(i.lo)
		if !i.indirect {
			cpu.r.PC = i.addr
			return true
		}
		i.state++
	case 2:
		i.addr += uint16(cpu.r.X)
		i.state++
	case 3:
		i.lo = cpu.psram.Read8(i.addr)
		i.state++
	case 4:
		i.hi = cpu.psram.Read8(i.addr + 1)
		cpu.r.PC = uint16(i.hi)<<8 | uint16(i.lo)
		return true
	}
	return false
}

func (i *JmpAbs) Reset(cpu *CPU) {
	i.state = 0
}

// all relative branch modes (29 instructions)
type Relative struct {
	am    AddressMode
	state int

	lo   byte
	addr uint16

	branchCondition func(*CPU, byte, uint16) bool
	shouldBranch    bool
}

func (i *Relative) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr := i.am.step(cpu)
		if next {
			i.lo = val
			i.addr = addr
			i.state++
		}
	case 1:
		i.shouldBranch = i.branchCondition(cpu, i.lo, i.addr)
		i.state++
	//FOOTGUN: state is reset to 2 if there is no addressing mode for regular branches
	case 2:
		if i.am == nil {
			i.shouldBranch = i.branchCondition(cpu, i.lo, i.addr)
		}
		i.lo = cpu.fetchByte()
		if i.shouldBranch {
			cpu.r.PC += uint16(int8(i.lo))
			i.state++
		} else {
			return true
		}
	case 3:
		i.state++
	case 4:
		i.state++
		return true
	}
	return false
}

func (i *Relative) Reset(cpu *CPU) {
	if i.am == nil {
		i.state = 2
	} else {
		i.state = 0
	}
}
