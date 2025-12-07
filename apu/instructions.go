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
	ret[0x02] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x01 }}}
	ret[0x22] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x02 }}}
	ret[0x42] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x04 }}}
	ret[0x62] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x08 }}}
	ret[0x82] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x10 }}}
	ret[0xA2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x20 }}}
	ret[0xC2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x40 }}}
	ret[0xE2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b | 0x80 }}}
	//CLR1
	ret[0x12] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFE }}}
	ret[0x32] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFD }}}
	ret[0x52] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xFB }}}
	ret[0x72] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xF7 }}}
	ret[0x92] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xEF }}}
	ret[0xB2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xDF }}}
	ret[0xD2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0xBF }}}
	ret[0xF2] = &ExecAndWrite8{skipExec: true, am: &DirectPage{io: READ_RAM, mode: BIT, bitOp: func(b byte) byte { return b & 0x7F }}}

	ret[0x4A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) && b)) }}
	ret[0x6A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) && !b)) }}
	ret[0xAA] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !b) }}
	ret[0x0A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) || b)) }, extraCycle: true}
	ret[0x2A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, !(c.r.hasFlag(FlagC) || !b)) }, extraCycle: true}
	ret[0x8A] = &MemBit{bitFunc: func(c *CPU, b bool) { c.r.setFlag(FlagC, c.r.hasFlag(FlagC) == b) }, extraCycle: true}
	ret[0xCA] = &MemBit{bitFuncWrite: func(c *CPU, _ byte) bool { return !c.r.hasFlag(FlagC) }, isWrite: true, extraCycle: true}
	ret[0xEA] = &MemBit{bitFuncWrite: func(_ *CPU, b byte) bool { return b != 0 }, isWrite: true, extraCycle: false}

	//TSET1 TCLR1
	ret[0x0E] = &ExecAndWrite8{func8: tset1, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x4E] = &ExecAndWrite8{func8: tclr1, am: &Absolute{io: READ_RAM, mode: DEFAULT}}

	//Stack Operations
	ret[0x2D] = &StackOp{isPush: true, getVal: func(c *CPU) *byte { return &c.r.A }}
	ret[0x4D] = &StackOp{isPush: true, getVal: func(c *CPU) *byte { return &c.r.X }}
	ret[0x6D] = &StackOp{isPush: true, getVal: func(c *CPU) *byte { return &c.r.Y }}
	ret[0x0D] = &StackOp{isPush: true, getVal: func(c *CPU) *byte { return &c.r.PSW }}

	ret[0xAE] = &StackOp{isPush: false, getVal: func(c *CPU) *byte { return &c.r.A }}
	ret[0xCE] = &StackOp{isPush: false, getVal: func(c *CPU) *byte { return &c.r.X }}
	ret[0xEE] = &StackOp{isPush: false, getVal: func(c *CPU) *byte { return &c.r.Y }}
	ret[0x8E] = &StackOp{isPush: false, getVal: func(c *CPU) *byte { return &c.r.PSW }}

	//8 bit shifts
	ret[0x1C] = &ExecAndWrite8{func8: asl, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x0B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: asl, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x1B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: asl, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x0C] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: asl, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x5C] = &ExecAndWrite8{func8: lsr, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x4B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: lsr, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x5B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: lsr, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x4C] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: lsr, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x3C] = &ExecAndWrite8{func8: rol, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x2B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: rol, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x3B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: rol, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x2C] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: rol, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x7C] = &ExecAndWrite8{func8: ror, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x6B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: ror, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x7B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: ror, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x6C] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: ror, am: &Absolute{io: READ_RAM, mode: DEFAULT}}

	ret[0x9F] = &XCN{}

	//8-bit inc/dec
	ret[0xBC] = &ExecAndWrite8{func8: inc, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0xAB] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: inc, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xBB] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: inc, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xAC] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: inc, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x3D] = &ExecAndWrite8{func8: inc, am: &AccessRegister{mode: REGISTER_X}}
	ret[0xFC] = &ExecAndWrite8{func8: inc, am: &AccessRegister{mode: REGISTER_Y}}

	ret[0x9C] = &ExecAndWrite8{func8: dec, am: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x8B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: dec, am: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x9B] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: dec, am: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x8C] = &ExecAndWrite8{executeImmediately: true, skipExec: true, func8: dec, am: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x1D] = &ExecAndWrite8{func8: dec, am: &AccessRegister{mode: REGISTER_X}}
	ret[0xDC] = &ExecAndWrite8{func8: dec, am: &AccessRegister{mode: REGISTER_Y}}

	//8-bit logical
	ret[0x28] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0x26] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0x24] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x34] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x25] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x35] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0x36] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0x27] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0x37] = &ExecAndWrite8x2Access{func8: and, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0x39] = &ExecAndWrite8x2Access{func8: and,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0x29] = &ExecAndWrite8x2Access{func8: and,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x38] = &ExecAndWrite8x2Access{func8: and,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}

	ret[0x08] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0x06] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0x04] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x14] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x05] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x15] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0x16] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0x07] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0x17] = &ExecAndWrite8x2Access{func8: or, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0x19] = &ExecAndWrite8x2Access{func8: or,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0x09] = &ExecAndWrite8x2Access{func8: or,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x18] = &ExecAndWrite8x2Access{func8: or,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}

	ret[0x48] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0x46] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0x44] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x54] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x45] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x55] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0x56] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0x47] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0x57] = &ExecAndWrite8x2Access{func8: eor, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0x59] = &ExecAndWrite8x2Access{func8: eor,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0x49] = &ExecAndWrite8x2Access{func8: eor,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x58] = &ExecAndWrite8x2Access{func8: eor,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}

	//8-bit arithmetic
	ret[0x88] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0x86] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0x84] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x94] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x85] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x95] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0x96] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0x87] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0x97] = &ExecAndWrite8x2Access{func8: adc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0x99] = &ExecAndWrite8x2Access{func8: adc,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0x89] = &ExecAndWrite8x2Access{func8: adc,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x98] = &ExecAndWrite8x2Access{func8: adc,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}

	ret[0xA8] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0xA6] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0xA4] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xB4] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xA5] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0xB5] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0xB6] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0xA7] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0xB7] = &ExecAndWrite8x2Access{func8: sbc, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0xB9] = &ExecAndWrite8x2Access{func8: sbc,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0xA9] = &ExecAndWrite8x2Access{func8: sbc,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xB8] = &ExecAndWrite8x2Access{func8: sbc,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}

	ret[0x68] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0x66] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0x64] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x74] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0x65] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x75] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0x76] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0x67] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0x77] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}
	ret[0x79] = &ExecAndWrite8x2Access{func8: cmp, skipWrite: true,
		am1: &DirectPage{io: READ_RAM, mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X, indexAndResolve: true}}
	ret[0x69] = &ExecAndWrite8x2Access{func8: cmp, skipWrite: true,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x78] = &ExecAndWrite8x2Access{func8: cmp, skipWrite: true,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xC8] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &Immediate{}}
	ret[0x3E] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x1E] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0xAD] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &Immediate{}}
	ret[0x7E] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x5E] = &ExecAndWrite8x2Access{func8: cmp, am1IsRegister: true, skipWrite: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}

	ret[0xFA] = &ExecAndWrite8x2Access{func8: movNoFlag, writeImmediately: true,
		am1: &DirectPage{io: READ_RAM, mode: DEFAULT}, am2: &DirectPage{io: WRITE_RAM, mode: DEFAULT}}
	ret[0x8F] = &ExecAndWrite8x2Access{func8: movNoFlag,
		am1: &Immediate{}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xBD] = &ExecAndWrite8x2Access{func8: movNoFlag, am1IsRegister: true,
		am1: &AccessRegister{mode: STACK_POINTER}, am2: &AccessRegister{mode: REGISTER_X}}
	ret[0x9D] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &AccessRegister{mode: STACK_POINTER}}
	ret[0xFD] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &AccessRegister{mode: ACCUMULATOR}}
	ret[0x5D] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &AccessRegister{mode: ACCUMULATOR}}
	ret[0xDD] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &AccessRegister{mode: REGISTER_Y}}
	ret[0x7D] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &AccessRegister{mode: REGISTER_X}}

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
		next, val, addr, _ := i.am.step(cpu)
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

// super speshul 5 cycle exception for XCN only
// shifting very hard
type XCN struct {
	state int

	addr uint16
}

func (i *XCN) Step(cpu *CPU) bool {
	switch i.state {
	case 0, 1, 2:
		i.state++
	case 3:
		val := cpu.r.A
		val = val<<4 | val>>4
		cpu.r.setFlag(FlagZ, val != 0)
		cpu.r.setFlag(FlagN, (val&0x80) == 0)

		cpu.r.A = val
		return true
	}
	return false
}
func (i *XCN) Reset() {
	i.state = 0
}

type ExecAndWrite8 struct {
	am    AddressMode
	state int

	skipExec           bool
	executeImmediately bool
	func8              InstructionFunc8

	lo   byte
	addr uint16
}

func (i *ExecAndWrite8) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr, register := i.am.step(cpu)
		if next {
			i.lo = val
			i.addr = addr
			if register != nil {
				*register = i.func8(cpu, i.lo, i.addr)
				return true
			}
			if i.executeImmediately {
				i.lo = i.func8(cpu, i.lo, i.addr)
			}
			if i.skipExec {
				i.state = 2
			} else {
				i.state = 1
			}
		}
	case 1:
		i.lo = i.func8(cpu, i.lo, i.addr)
		i.state++
	case 2:
		cpu.psram.Write8(i.addr, i.lo)
		return true
	}
	return false
}

func (i *ExecAndWrite8) Reset() {
	i.state = 0
	i.am.reset()
}

type ExecAndWrite8x2Access struct {
	am1, am2 AddressMode
	state    int

	func8 InstructionFunc8x2

	val1, val2               byte
	addr1, addr2             uint16
	regPointer1, regPointer2 *byte

	next, am1IsRegister, skipWrite, writeImmediately bool
}

func (i *ExecAndWrite8x2Access) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.next, i.val1, i.addr1, _ = i.am1.step(cpu)
		if i.next {
			i.state++
		}
	//another footgun. when its a register read its a cycle shorter
	case 1:
		i.next, i.val2, i.addr2, _ = i.am2.step(cpu)
		if !i.next {
			return false
		}
		if i.am1IsRegister {
			_, i.val1, _, i.regPointer1 = i.am1.step(cpu) //next is always true
		}
		if i.regPointer1 != nil {
			ret := i.func8(cpu, i.val1, i.val2, i.addr1, i.addr2)
			if !i.skipWrite {
				*i.regPointer1 = ret
			}
			return true
		} else {
			i.val2 = i.func8(cpu, i.val2, i.val1, i.addr2, i.addr1)
			if i.writeImmediately { //mov instructions skip a write cycle sometimes
				cpu.psram.Write8(i.addr2, i.val2)
				return true
			}
		}
		i.state++
	case 2:
		if !i.skipWrite {
			cpu.psram.Write8(i.addr2, i.val2)
		}
		return true
	}
	return false
}

func (i *ExecAndWrite8x2Access) Reset() {
	if i.am1IsRegister {
		i.state = 1
	} else {
		i.state = 0
	}
	i.am1.reset()
	i.am2.reset()
}

type StackOp struct {
	state int

	getVal func(*CPU) *byte
	isPush bool

	valPointer *byte
	addr       uint16
}

func (i *StackOp) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.valPointer = i.getVal(cpu)
		i.state++
	case 1:
		if i.isPush {
			cpu.PushByte(*i.valPointer)
		}
		i.state++
	case 2:
		if !i.isPush {
			*i.valPointer = cpu.PopByte()
		}
		return true
	}
	return false
}

func (i *StackOp) Reset() {
	i.state = 0
}
