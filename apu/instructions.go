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

	//MOV
	//8-bit reg->reg, mem->mem
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

	//8-bit write
	ret[0xC6] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0xAF] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: WRITE_RAM, mode: REGISTER_X, autoIncrement: 1}}
	ret[0xC4] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xD4] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xC5] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0xD5] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0xD6] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0xC7] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0xD7] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: ACCUMULATOR}, am1: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED_LAST}}
	ret[0xD8] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_X}, am1: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xD9] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_X}, am1: &DirectPage{io: READ_RAM, mode: Y_INDEXED}}
	ret[0xC9] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_X}, am1: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0xCB] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_Y}, am1: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xDB] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_Y}, am1: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xCC] = &ExecAndWrite8x2Access{func8: movNoFlagInverse, writeAddr1: true, writeImmediately: true,
		am2: &AccessRegister{mode: REGISTER_Y}, am1: &Absolute{io: READ_RAM, mode: DEFAULT}}

	//8-bit read
	ret[0xE8] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Immediate{}}
	ret[0xE6] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0xBF] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true, incrementXafter: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: REGISTER_X}}
	ret[0xE4] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xF4] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xE5] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0xF5] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: X_INDEXED}}
	ret[0xF6] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &Absolute{io: READ_RAM, mode: Y_INDEXED}}
	ret[0xE7] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDEXED_INDIRECT}}
	ret[0xF7] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: ACCUMULATOR}, am2: &DirectPage{io: READ_RAM, mode: INDIRECT_INDEXED}}

	ret[0xCD] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &Immediate{}}
	ret[0xF8] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xF9] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &DirectPage{io: READ_RAM, mode: Y_INDEXED}}
	ret[0xE9] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_X}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}
	ret[0x8D] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &Immediate{}}
	ret[0xEB] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xFB] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &DirectPage{io: READ_RAM, mode: X_INDEXED}}
	ret[0xEC] = &ExecAndWrite8x2Access{func8: mov, am1IsRegister: true,
		am1: &AccessRegister{mode: REGISTER_Y}, am2: &Absolute{io: READ_RAM, mode: DEFAULT}}

	//mul/div
	ret[0xCF] = &Mul{}
	ret[0x9E] = &Div{}

	ret[0xDF] = &DecimalAdjust{iFunc: daAddition}
	ret[0xBE] = &DecimalAdjust{iFunc: daSubtraction}

	//16-bit addw subw cmpw
	ret[0x7A] = &Exec16{func16: addW, am: DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x9A] = &Exec16{func16: subW, am: DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0x5A] = &Exec16{func16: cmpW, am: DirectPage{io: READ_RAM, mode: DEFAULT}, skipIdleCycle: true}
	//incW decW
	ret[0x1A] = &IncWDecW{am: DirectPage{io: READ_RAM, mode: DEFAULT}, amount: 0xFF}
	ret[0x3A] = &IncWDecW{am: DirectPage{io: READ_RAM, mode: DEFAULT}, amount: 0x1}

	//16-bit mov
	ret[0xBA] = &Exec16{func16: mov16, am: DirectPage{io: READ_RAM, mode: DEFAULT}}
	ret[0xDA] = &ExecAndWrite16{am: DirectPage{io: READ_RAM, mode: DEFAULT}} //mov16 the other way requires a new struct

	//PSW operations
	ret[0x60] = &TwoCycleImplied{iFunc: func(c *CPU) { c.r.setFlag(FlagC, true) }}
	ret[0x80] = &TwoCycleImplied{iFunc: func(c *CPU) { c.r.setFlag(FlagC, false) }}
	ret[0xE0] = &TwoCycleImplied{iFunc: func(c *CPU) { c.r.setFlag(FlagH, true); c.r.setFlag(FlagV, true) }}
	ret[0x20] = &TwoCycleImplied{iFunc: func(c *CPU) { c.r.setFlag(FlagP, true) }}
	ret[0x40] = &TwoCycleImplied{iFunc: func(c *CPU) { c.r.setFlag(FlagP, false) }}
	ret[0xED] = &DecimalAdjust{iFunc: func(c *CPU) { c.r.PSW ^= FlagC }}
	ret[0xA0] = &DecimalAdjust{iFunc: func(c *CPU) { c.r.setFlag(FlagI, false) }}
	ret[0xC0] = &DecimalAdjust{iFunc: func(c *CPU) { c.r.setFlag(FlagI, true) }}

	//subroutine instructions
	//TCALL/BRK
	ret[0x01] = &TcallBrk{vector: 0xFFDE}
	ret[0x11] = &TcallBrk{vector: 0xFFDC}
	ret[0x21] = &TcallBrk{vector: 0xFFDA}
	ret[0x31] = &TcallBrk{vector: 0xFFD8}
	ret[0x41] = &TcallBrk{vector: 0xFFD6}
	ret[0x51] = &TcallBrk{vector: 0xFFD4}
	ret[0x61] = &TcallBrk{vector: 0xFFD2}
	ret[0x71] = &TcallBrk{vector: 0xFFD0}
	ret[0x81] = &TcallBrk{vector: 0xFFCE}
	ret[0x91] = &TcallBrk{vector: 0xFFCC}
	ret[0xA1] = &TcallBrk{vector: 0xFFCA}
	ret[0xB1] = &TcallBrk{vector: 0xFFC8}
	ret[0xC1] = &TcallBrk{vector: 0xFFC6}
	ret[0xD1] = &TcallBrk{vector: 0xFFC4}
	ret[0xE1] = &TcallBrk{vector: 0xFFC2}
	ret[0xF1] = &TcallBrk{vector: 0xFFC0}
	ret[0x0F] = &TcallBrk{vector: 0xFFDE, isBrk: true}
	//ret/reti
	ret[0x6F] = &RetReti{}
	ret[0x7F] = &RetReti{isReti: true}
	//call/pcall
	ret[0x3F] = &CallPcall{}
	ret[0x4F] = &CallPcall{isPcall: true}

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

	val1, val2   byte
	addr1, addr2 uint16
	regPointer1  *byte

	next, am1IsRegister, skipWrite, writeImmediately, writeAddr1 bool
	incrementXafter                                              bool //for one very special instruction
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
			ret := i.func8(cpu, i.val1, i.val2, i.addr1, i.addr2)
			if !i.skipWrite {
				*i.regPointer1 = ret
			}
			if !i.incrementXafter {
				return true
			} else {
				i.state = 3
				return false
			}
		} else {
			i.val2 = i.func8(cpu, i.val2, i.val1, i.addr2, i.addr1)
			if i.writeImmediately { //mov instructions skip a write cycle sometimes
				if i.writeAddr1 {
					cpu.psram.Write8(i.addr1, i.val2)
				} else {
					cpu.psram.Write8(i.addr2, i.val2)
				}
				return true
			}
		}
		i.state++
	case 2:
		if !i.skipWrite {
			if i.writeAddr1 {
				cpu.psram.Write8(i.addr1, i.val2)
			} else {
				cpu.psram.Write8(i.addr2, i.val2)
			}
		}
		return true
	case 3:
		cpu.r.X++
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

type Mul struct {
	state int
}

func (i *Mul) Step(cpu *CPU) bool {
	switch i.state {
	case 0, 1, 2, 3, 4, 5, 6:
		i.state++
	case 7:
		ya := uint16(cpu.r.Y) * uint16(cpu.r.A)
		y := byte(ya >> 8)
		cpu.r.Y, cpu.r.A = y, byte(ya)
		cpu.r.setFlag(FlagZ, y != 0)
		cpu.r.setFlag(FlagN, (y&0x80) == 0)
		return true
	}
	return false
}

func (i *Mul) Reset() {
	i.state = 0
}

type Div struct {
	state int
}

// thx mame
// Y <- YA % X and A <- YA / X
func (i *Div) Step(cpu *CPU) bool {
	switch i.state {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
		i.state++
	case 10:
		ya := uint32(cpu.r.Y)<<8 | uint32(cpu.r.A)
		x := uint32(cpu.r.X) << 9
		cpu.r.setFlag(FlagH, (cpu.r.Y&0xF) < (cpu.r.X&0xF))
		for range 9 {
			ya <<= 1
			if ya&0x20000 > 0 {
				ya = ya&0x1FFFF | 1
			}
			if ya >= x {
				ya ^= 1
			}
			if ya&1 > 0 {
				ya = ((ya - x) & 0x1FFFF)
			}
		}
		cpu.r.setFlag(FlagV, ya&0x100 == 0)
		ya = (((ya >> 9) & 0xFF) << 8) + (ya & 0xFF)
		cpu.r.Y = byte(ya >> 8)
		cpu.r.A = byte(ya)

		cpu.r.setFlag(FlagZ, byte(ya) != 0)
		cpu.r.setFlag(FlagN, (byte(ya)&0x80) == 0)
		return true
	}
	return false
}

func (i *Div) Reset() {
	i.state = 0
}

// TODO make this into a more generic 3cycleImplied template if possible
type DecimalAdjust struct {
	state int

	iFunc ImpliedFunc
}

func (i *DecimalAdjust) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.iFunc(cpu)
		return true
	}
	return false
}

func (i *DecimalAdjust) Reset() {
	i.state = 0
}

type TwoCycleImplied struct {
	iFunc ImpliedFunc
}

func (i *TwoCycleImplied) Step(cpu *CPU) bool {
	i.iFunc(cpu)
	return true
}

func (i *TwoCycleImplied) Reset() {
}

type Exec16 struct {
	state int
	am    DirectPage

	addr          uint16
	lo            byte
	func16        InstructionFunc16
	skipIdleCycle bool
}

func (i *Exec16) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr, _ := i.am.step(cpu)
		if next {
			i.lo, i.addr = val, addr
			if i.skipIdleCycle {
				i.state = 2
			} else {
				i.state = 1
			}
		}
	case 1:
		i.state++
	case 2:
		hi := cpu.psram.Read8(uint16(cpu.r.getDirectPageNum())<<8 | ((i.addr + 1) & 0xFF))
		ya := uint16(cpu.r.Y)<<8 | uint16(cpu.r.A)
		word := uint16(hi)<<8 | uint16(i.lo)

		i.func16(cpu, ya, word)
		return true
	}
	return false
}

func (i *Exec16) Reset() {
	i.state = 0
	i.am.reset()
}

// basically 1 very special 16 bit mov instruction
type ExecAndWrite16 struct {
	state int
	am    DirectPage

	addr uint16
	lo   byte
}

func (i *ExecAndWrite16) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr, _ := i.am.step(cpu)
		if next {
			i.lo, i.addr = val, addr
			i.state = 1
		}
	case 1:
		cpu.psram.Write8(i.addr, cpu.r.A)
		i.state++
	case 2:
		cpu.psram.Write8(uint16(cpu.r.getDirectPageNum())<<8|((i.addr+1)&0xFF), cpu.r.Y)
		return true
	}
	return false
}

func (i *ExecAndWrite16) Reset() {
	i.state = 0
	i.am.reset()
}

type IncWDecW struct {
	state int
	am    DirectPage

	addr   uint16
	lo, hi byte
	carry  bool
	amount byte
}

func (i *IncWDecW) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		next, val, addr, _ := i.am.step(cpu)
		if next {
			i.lo, i.addr = val, addr
			i.state++
		}
	case 1:
		result := uint16(i.lo) + uint16(i.amount)
		if i.amount == 1 {
			i.carry = result > 0xFF
		} else {
			i.carry = i.lo == 0
		}

		cpu.psram.Write8(i.addr, byte(result))
		i.state++
	case 2:
		i.addr = uint16(cpu.r.getDirectPageNum())<<8 | ((i.addr + 1) & 0xFF)
		i.hi = cpu.psram.Read8(i.addr)
		word := uint16(i.hi)<<8 | uint16(i.lo)

		cpu.r.setFlag(FlagZ, word != 0)
		cpu.r.setFlag(FlagN, (word&0x8000) == 0)
		i.state++
	case 3:
		if i.carry {
			cpu.psram.Write8(i.addr, i.hi+i.amount)
		} else {
			cpu.psram.Write8(i.addr, i.hi)
		}
		return true
	}
	return false
}

func (i *IncWDecW) Reset() {
	i.state = 0
	i.am.reset()
}

type TcallBrk struct {
	state int

	vector uint16
	isBrk  bool
}

func (i *TcallBrk) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		if i.isBrk {
			i.state = 2
		} else {
			cpu.psram.Read8(cpu.r.PC) //dummy read
			i.state = 1
		}
	case 1:
		i.state++
	case 2:
		cpu.PushByte(byte(cpu.r.PC >> 8))
		i.state++
	case 3:
		cpu.PushByte(byte(cpu.r.PC))
		i.state++
	case 4:
		if i.isBrk {
			cpu.PushByte(cpu.r.PSW)
			cpu.r.setFlag(FlagB, false)
			cpu.r.setFlag(FlagI, true)
			i.state = 7
		} else {
			i.state = 5
		}
	case 5:
		cpu.r.PC = (cpu.r.PC & 0xFF00) | uint16(cpu.psram.Read8(i.vector))
		i.state++
	case 6:
		cpu.r.PC = (cpu.r.PC & 0xFF) | uint16(cpu.psram.Read8(i.vector+1))<<8
		return true
	case 7:
		i.state = 5
	}
	return false
}

func (i *TcallBrk) Reset() {
	i.state = 0
}

type RetReti struct {
	state int

	isReti bool
	pcl    byte
}

func (i *RetReti) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		if i.isReti {
			cpu.r.PSW = cpu.PopByte()
		}
		i.state++
	case 3:
		i.pcl = cpu.PopByte()
		i.state++
	case 4:
		cpu.r.PC = (uint16(cpu.PopByte())<<8 | uint16(i.pcl))
		return true
	}
	return false
}

func (i *RetReti) Reset() {
	if i.isReti {
		i.state = 0
	} else {
		i.state = 1
	}
}

type CallPcall struct {
	state int

	isPcall  bool
	pcl, pch byte
}

func (i *CallPcall) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.pcl = cpu.fetchByte()
		if i.isPcall {
			i.pch = 0xFF
			i.state = 2
		} else {
			i.state = 1
		}
	case 1:
		i.pch = cpu.fetchByte()
		i.state++
	case 2:
		i.state++
	case 3:
		cpu.PushByte(byte(cpu.r.PC >> 8))
		i.state++
	case 4:
		cpu.PushByte(byte(cpu.r.PC))
		i.state++
	case 5:
		cpu.r.PC = uint16(i.pch)<<8 | uint16(i.pcl)
		if i.isPcall {
			return true
		}
		i.state++
	case 6:
		return true
	}
	return false
}

func (i *CallPcall) Reset() {
	i.state = 0
}
