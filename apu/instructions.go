package apu

// Instruction represents a single CPU instruction, executed one cycle at a time.
type Instruction interface {
	// Step performs one cycle of the instruction's execution.
	// It returns true if the instruction is complete, false otherwise.
	Step(cpu *CPU) bool
	Reset()
}

func NewInstructionMap() []Instruction {
	ret := make([]Instruction, 0x100)

	ret[0x5F] = &JmpAbs{indirect: false}
	ret[0x1F] = &JmpAbs{indirect: true}

	//BBC
	branchIfEqual := func(_ *CPU, b byte, _ uint16) bool { return b == 0 }
	ret[0x13] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x01 }}}
	ret[0x33] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x02 }}}
	ret[0x53] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x04 }}}
	ret[0x73] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x08 }}}
	ret[0x93] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x10 }}}
	ret[0xB3] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x20 }}}
	ret[0xD3] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x40 }}}
	ret[0xF3] = &Relative{branchCondition: branchIfEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x80 }}}
	//BBS
	branchIfNotEqual := func(_ *CPU, b byte, _ uint16) bool { return b != 0 }
	ret[0x03] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x01 }}}
	ret[0x23] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x02 }}}
	ret[0x43] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x04 }}}
	ret[0x63] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x08 }}}
	ret[0x83] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x10 }}}
	ret[0xA3] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x20 }}}
	ret[0xC3] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x40 }}}
	ret[0xE3] = &Relative{branchCondition: branchIfNotEqual,
		am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x80 }}}
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

	//SET1
	ret[0x02] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x01 }}}
	ret[0x22] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x02 }}}
	ret[0x42] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x04 }}}
	ret[0x62] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x08 }}}
	ret[0x82] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x10 }}}
	ret[0xA2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x20 }}}
	ret[0xC2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x40 }}}
	ret[0xE2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x80 }}}
	//CLR1
	ret[0x12] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFE }}}
	ret[0x32] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFD }}}
	ret[0x52] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFB }}}
	ret[0x72] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xF7 }}}
	ret[0x92] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xEF }}}
	ret[0xB2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xDF }}}
	ret[0xD2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xBF }}}
	ret[0xF2] = &SetClr1{am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x7F }}}

	ret[0x4A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) && b)) }}
	ret[0x6A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) && !b)) }}
	ret[0xAA] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !b) }}
	ret[0x0A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) || b)) }, extraCycle: true}
	ret[0x2A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) || !b)) }, extraCycle: true}
	ret[0x8A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, c.r.hasFlag(FlagC) == b) }, extraCycle: true}
	ret[0xCA] = &MemBit{bitFuncWrite: func(c *CPU, _ byte) bool { return !c.r.hasFlag(FlagC) }, isWrite: true, extraCycle: true}
	ret[0xEA] = &MemBit{bitFuncWrite: func(_ *CPU, b byte) bool { return b != 0 }, isWrite: true, extraCycle: false}

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

func (i *JmpAbs) Reset() {
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

func (i *Relative) Reset() {
	if i.am == nil {
		i.state = 2
	} else {
		i.state = 0
		i.am.reset()
	}
}

type SetClr1 struct {
	am    AddressMode
	state int

	lo   byte
	addr uint16
}

func (i *SetClr1) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr := i.am.step(cpu)
		if next {
			i.lo = val
			i.addr = addr
			i.state++
		}
	case 1:
		cpu.psram.Write8(i.addr, i.lo)
		return true
	}
	return false
}
func (i *SetClr1) Reset() {
	i.state = 0
	i.am.reset()
}

type MemBit struct {
	state int

	lo, bit byte
	addr    uint16

	isWrite    bool
	extraCycle bool

	bitFunc      func(*CPU, bool)
	bitFuncWrite func(*CPU, byte) bool
}

func (i *MemBit) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lo = cpu.fetchByte()
		i.state++
	case 1:
		hi := cpu.fetchByte()
		i.addr = (uint16(hi)<<8 | uint16(i.lo)) & 0x1FFF
		i.bit = hi >> 5
		i.state++
	case 2:
		i.lo = cpu.psram.Read8(i.addr)
		if i.extraCycle {
			i.state = 3
			return false
		}
		if i.isWrite {
			i.state = 4
			return false
		}
		hasBit := i.lo&(1<<i.bit) != 0
		i.bitFunc(cpu, hasBit)
		return true
	case 3:
		if i.isWrite {
			i.state = 4
			return false
		}
		hasBit := i.lo&(1<<i.bit) != 0
		i.bitFunc(cpu, hasBit)
		return true
	case 4:
		mask := byte(1 << i.bit)
		if i.bitFuncWrite(cpu, i.lo&mask) {
			cpu.psram.Write8(i.addr, i.lo&^mask)
		} else {
			cpu.psram.Write8(i.addr, i.lo|mask)
		}
		return true
	}
	return false
}
func (i *MemBit) Reset() {
	i.state = 0
}
