package cpu

// Instruction represents a single CPU instruction, executed one cycle at a time.
type Instruction interface {
	// Step performs one cycle of the instruction's execution.
	// It returns true if the instruction is complete, false otherwise.
	Step(cpu *CPU) bool
	Reset(cpu *CPU)
}

func NewHWInterruptMap() []Instruction {
	ret := make([]Instruction, 4)
	ret[irqId] = &NmiIrqSequence{eAddress: 0x00FFFE, nAddress: 0x00FFEE}
	ret[nmiId] = &NmiIrqSequence{eAddress: 0x00FFFA, nAddress: 0x00FFEA}
	ret[abortId] = &AbortSequence{eAddress: 0x00FFF8, nAddress: 0x00FFE8}
	ret[resetId] = &ResetSequence{eAddress: 0x00FFFC}

	return ret
}

func NewInstructionMap() []Instruction {
	ret := make([]Instruction, 256)

	ret[0x4C] = &JMP_Abs{}
	ret[0x5C] = &JMP_Long{}
	ret[0x6C] = &JMP_AbsIndirect{}
	ret[0x7C] = &JMP_AbsIndexedIndirect{}
	ret[0xDC] = &JMP_AbsLong{}
	ret[0xFC] = &JSR_AbsIndexedIndirect{}
	ret[0x20] = &JSR_Abs{}
	ret[0x22] = &JSL{}

	ret[0x40] = &RTI{}
	ret[0x6B] = &RtsRtl{long: true}
	ret[0x60] = &RtsRtl{long: false}

	ret[0x82] = &BRL{}

	ret[0x80] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return true }}                  //BRA or branch always
	ret[0x10] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagN) }} //BPL or branch if positive
	ret[0x30] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagN) }}  //BMI or branch if not positive
	ret[0x90] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagC) }} //BCC or branch if no carry
	ret[0xB0] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagC) }}  //BCS or branch if carry
	ret[0xF0] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagZ) }}  //BEQ or branch if zero
	ret[0xD0] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagZ) }} //BNE or branch if not zero
	ret[0x50] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagV) }} //BVC or branch if not overflow
	ret[0x70] = &OneByteBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagV) }}  //BVS or branch if overflow

	// I00 represents the BRK or break (software interrupt) instruction
	ret[0x00] = &softwareInterrupt{eAddress: 0x00FFFE, nAddress: 0x00FFE6}
	// I02 represents the COP (software interrupt) instruction
	ret[0x02] = &softwareInterrupt{eAddress: 0x00FFF4, nAddress: 0x00FFE4}

	// CLC CLD CLI CLV SEC SED SEI
	ret[0x18] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.r.setFlag(FlagC, true) }}
	ret[0xD8] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.r.setFlag(FlagD, true) }}
	ret[0x58] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) {
		cpu.previousIFlag = int(cpu.r.P & FlagI)
		cpu.r.setFlag(FlagI, true)
	}}
	ret[0xB8] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.r.setFlag(FlagV, true) }}
	ret[0x38] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.r.setFlag(FlagC, false) }}
	ret[0xF8] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.r.setFlag(FlagD, false) }}
	ret[0x78] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) {
		cpu.previousIFlag = int(cpu.r.P & FlagI)
		cpu.r.setFlag(FlagI, false)
	}}

	ret[0xC2] = &RepSep{reset: true}  //rep
	ret[0xE2] = &RepSep{reset: false} //sep

	ret[0xFB] = &TwoCycleImplied{instructionFunc: xce}

	//STP/WAI
	ret[0xDB] = &StpWai{executionState: stopState}
	ret[0xCB] = &StpWai{executionState: waitState}

	ret[0xEB] = &XBA{}

	//WDM/NOP instructions
	ret[0xEA] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) {}}
	ret[0x42] = &TwoCycleImplied{instructionFunc: func(cpu *CPU) { cpu.fetchByte() }}

	//the shift and rotate instructions
	ret[0x0A] = &Accumulator{instructionFunc: asl}
	ret[0x4A] = &Accumulator{instructionFunc: lsr}
	ret[0x6A] = &Accumulator{instructionFunc: ror}
	ret[0x2A] = &Accumulator{instructionFunc: rol}

	ret[0x46] = NewUmbrellaWrite(lsr, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x26] = NewUmbrellaWrite(rol, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x06] = NewUmbrellaWrite(asl, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x66] = NewUmbrellaWrite(ror, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)

	ret[0x56] = NewUmbrellaWrite(lsr, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x36] = NewUmbrellaWrite(rol, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x16] = NewUmbrellaWrite(asl, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x76] = NewUmbrellaWrite(ror, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)

	ret[0x4E] = NewUmbrellaWrite(lsr, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x2E] = NewUmbrellaWrite(rol, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x0E] = NewUmbrellaWrite(asl, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x6E] = NewUmbrellaWrite(ror, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)

	ret[0x5E] = NewUmbrellaWrite(lsr, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x3E] = NewUmbrellaWrite(rol, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x1E] = NewUmbrellaWrite(asl, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0x7E] = NewUmbrellaWrite(ror, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)

	//Test and Set/Test and Reset bits
	ret[0x1C] = NewUmbrellaWrite(trb, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x0C] = NewUmbrellaWrite(tsb, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x14] = NewUmbrellaWrite(trb, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0x04] = NewUmbrellaWrite(tsb, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)

	//all 14 sta instructions
	ret[0x81] = NewUmbrellaWrite(sta, &Direct{mode: INDEXED_INDIRECT}, true, true, false, false, is8BitM)
	ret[0x83] = NewUmbrellaWrite(sta, &StackS{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x85] = NewUmbrellaWrite(sta, &Direct{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x87] = NewUmbrellaWrite(sta, &Direct{mode: INDIRECT_LONG}, true, true, true, false, is8BitM)
	ret[0x8D] = NewUmbrellaWrite(sta, &Absolute{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x8F] = NewUmbrellaWrite(sta, &Long{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x91] = NewUmbrellaWrite(sta, &Direct{mode: INDIRECT_INDEXED}, true, false, false, false, is8BitM)
	ret[0x92] = NewUmbrellaWrite(sta, &Direct{mode: INDIRECT}, true, true, false, false, is8BitM)
	ret[0x93] = NewUmbrellaWrite(sta, &StackS{mode: INDIRECT_INDEXED}, true, true, true, false, is8BitM)
	ret[0x95] = NewUmbrellaWrite(sta, &Direct{mode: BASE_MODE_X}, true, true, true, false, is8BitM)
	ret[0x97] = NewUmbrellaWrite(sta, &Direct{mode: INDIRECT_LONG_INDEXED}, true, true, false, false, is8BitM)
	ret[0x99] = NewUmbrellaWrite(sta, &Absolute{mode: BASE_MODE_Y}, true, true, true, false, is8BitM)
	ret[0x9D] = NewUmbrellaWrite(sta, &Absolute{mode: BASE_MODE_X}, true, true, true, false, is8BitM)
	ret[0x9F] = NewUmbrellaWrite(sta, &Long{mode: BASE_MODE_X}, true, true, true, false, is8BitM)

	//all 3 stx instructions
	ret[0x86] = NewUmbrellaWrite(stx, &Direct{mode: BASE_MODE}, true, true, true, false, is8BitX)
	ret[0x8E] = NewUmbrellaWrite(stx, &Absolute{mode: BASE_MODE}, true, true, true, false, is8BitX)
	ret[0x96] = NewUmbrellaWrite(stx, &Direct{mode: BASE_MODE_Y}, true, true, true, false, is8BitX)
	//all 3 sty instructions
	ret[0x84] = NewUmbrellaWrite(sty, &Direct{mode: BASE_MODE}, true, true, true, false, is8BitX)
	ret[0x8C] = NewUmbrellaWrite(sty, &Absolute{mode: BASE_MODE}, true, true, true, false, is8BitX)
	ret[0x94] = NewUmbrellaWrite(sty, &Direct{mode: BASE_MODE_X}, true, true, true, false, is8BitX)

	//all 4 stz instructions
	ret[0x64] = NewUmbrellaWrite(stz, &Direct{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x74] = NewUmbrellaWrite(stz, &Direct{mode: BASE_MODE_X}, true, true, true, false, is8BitM)
	ret[0x9C] = NewUmbrellaWrite(stz, &Absolute{mode: BASE_MODE}, true, true, true, false, is8BitM)
	ret[0x9E] = NewUmbrellaWrite(stz, &Absolute{mode: BASE_MODE_X}, true, true, true, false, is8BitM)

	//all 15 LDA instructions
	ret[0xA1] = NewUmbrellaRead(lda, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0xA3] = NewUmbrellaRead(lda, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0xA5] = NewUmbrellaRead(lda, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0xA7] = NewUmbrellaRead(lda, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0xA9] = NewUmbrellaRead(lda, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0xAD] = NewUmbrellaRead(lda, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0xAF] = NewUmbrellaRead(lda, &Long{mode: BASE_MODE}, is8BitM)
	ret[0xB1] = NewUmbrellaRead(lda, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xB2] = NewUmbrellaRead(lda, &Direct{mode: INDIRECT}, is8BitM)
	ret[0xB3] = NewUmbrellaRead(lda, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xB5] = NewUmbrellaRead(lda, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0xB7] = NewUmbrellaRead(lda, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0xB9] = NewUmbrellaRead(lda, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0xBD] = NewUmbrellaRead(lda, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0xBF] = NewUmbrellaRead(lda, &Long{mode: BASE_MODE_X}, is8BitM)

	//all 5 LDX instructions
	ret[0xA2] = NewUmbrellaRead(ldx, &Immediate{mode: CHECK_PARENT}, is8BitX)
	ret[0xA6] = NewUmbrellaRead(ldx, &Direct{mode: BASE_MODE}, is8BitX)
	ret[0xAE] = NewUmbrellaRead(ldx, &Absolute{mode: BASE_MODE}, is8BitX)
	ret[0xB6] = NewUmbrellaRead(ldx, &Direct{mode: BASE_MODE_Y}, is8BitX)
	ret[0xBE] = NewUmbrellaRead(ldx, &Absolute{mode: BASE_MODE_Y}, is8BitX)

	//all 5 LDY instructions
	ret[0xA0] = NewUmbrellaRead(ldy, &Immediate{mode: CHECK_PARENT}, is8BitX)
	ret[0xA4] = NewUmbrellaRead(ldy, &Direct{mode: BASE_MODE}, is8BitX)
	ret[0xAC] = NewUmbrellaRead(ldy, &Absolute{mode: BASE_MODE}, is8BitX)
	ret[0xB4] = NewUmbrellaRead(ldy, &Direct{mode: BASE_MODE_X}, is8BitX)
	ret[0xBC] = NewUmbrellaRead(ldy, &Absolute{mode: BASE_MODE_X}, is8BitX)

	//test BITs
	ret[0x89] = NewUmbrellaRead(bit_imm, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0x24] = NewUmbrellaRead(bit, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0x2C] = NewUmbrellaRead(bit, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0x34] = NewUmbrellaRead(bit, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0x3C] = NewUmbrellaRead(bit, &Absolute{mode: BASE_MODE_X}, is8BitM)

	//bitwise AND all 15
	ret[0x21] = NewUmbrellaRead(and, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0x23] = NewUmbrellaRead(and, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0x25] = NewUmbrellaRead(and, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0x27] = NewUmbrellaRead(and, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0x29] = NewUmbrellaRead(and, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0x2D] = NewUmbrellaRead(and, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0x2F] = NewUmbrellaRead(and, &Long{mode: BASE_MODE}, is8BitM)
	ret[0x31] = NewUmbrellaRead(and, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x32] = NewUmbrellaRead(and, &Direct{mode: INDIRECT}, is8BitM)
	ret[0x33] = NewUmbrellaRead(and, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x35] = NewUmbrellaRead(and, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0x37] = NewUmbrellaRead(and, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0x39] = NewUmbrellaRead(and, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0x3D] = NewUmbrellaRead(and, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0x3F] = NewUmbrellaRead(and, &Long{mode: BASE_MODE_X}, is8BitM)

	//bitwise EOR all 15
	ret[0x41] = NewUmbrellaRead(eor, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0x43] = NewUmbrellaRead(eor, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0x45] = NewUmbrellaRead(eor, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0x47] = NewUmbrellaRead(eor, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0x49] = NewUmbrellaRead(eor, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0x4D] = NewUmbrellaRead(eor, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0x4F] = NewUmbrellaRead(eor, &Long{mode: BASE_MODE}, is8BitM)
	ret[0x51] = NewUmbrellaRead(eor, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x52] = NewUmbrellaRead(eor, &Direct{mode: INDIRECT}, is8BitM)
	ret[0x53] = NewUmbrellaRead(eor, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x55] = NewUmbrellaRead(eor, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0x57] = NewUmbrellaRead(eor, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0x59] = NewUmbrellaRead(eor, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0x5D] = NewUmbrellaRead(eor, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0x5F] = NewUmbrellaRead(eor, &Long{mode: BASE_MODE_X}, is8BitM)

	//bitwise ORA all 15
	ret[0x01] = NewUmbrellaRead(ora, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0x03] = NewUmbrellaRead(ora, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0x05] = NewUmbrellaRead(ora, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0x07] = NewUmbrellaRead(ora, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0x09] = NewUmbrellaRead(ora, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0x0D] = NewUmbrellaRead(ora, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0x0F] = NewUmbrellaRead(ora, &Long{mode: BASE_MODE}, is8BitM)
	ret[0x11] = NewUmbrellaRead(ora, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x12] = NewUmbrellaRead(ora, &Direct{mode: INDIRECT}, is8BitM)
	ret[0x13] = NewUmbrellaRead(ora, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x15] = NewUmbrellaRead(ora, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0x17] = NewUmbrellaRead(ora, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0x19] = NewUmbrellaRead(ora, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0x1D] = NewUmbrellaRead(ora, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0x1F] = NewUmbrellaRead(ora, &Long{mode: BASE_MODE_X}, is8BitM)

	//DECrement
	ret[0x3A] = &Accumulator{instructionFunc: dec}
	ret[0xC6] = NewUmbrellaWrite(dec, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0xCE] = NewUmbrellaWrite(dec, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0xD6] = NewUmbrellaWrite(dec, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0xDE] = NewUmbrellaWrite(dec, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0xCA] = &TwoCycleImplied{instructionFunc: decX}
	ret[0x88] = &TwoCycleImplied{instructionFunc: decY}

	//INCrement
	ret[0x1A] = &Accumulator{instructionFunc: inc}
	ret[0xE6] = NewUmbrellaWrite(inc, &Direct{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0xEE] = NewUmbrellaWrite(inc, &Absolute{mode: BASE_MODE}, false, false, false, false, is8BitM)
	ret[0xF6] = NewUmbrellaWrite(inc, &Direct{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0xFE] = NewUmbrellaWrite(inc, &Absolute{mode: BASE_MODE_X}, false, false, false, false, is8BitM)
	ret[0xE8] = &TwoCycleImplied{instructionFunc: incX}
	ret[0xC8] = &TwoCycleImplied{instructionFunc: incY}

	//CoMPare all 15
	ret[0xC1] = NewUmbrellaRead(cmp, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0xC3] = NewUmbrellaRead(cmp, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0xC5] = NewUmbrellaRead(cmp, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0xC7] = NewUmbrellaRead(cmp, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0xC9] = NewUmbrellaRead(cmp, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0xCD] = NewUmbrellaRead(cmp, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0xCF] = NewUmbrellaRead(cmp, &Long{mode: BASE_MODE}, is8BitM)
	ret[0xD1] = NewUmbrellaRead(cmp, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xD2] = NewUmbrellaRead(cmp, &Direct{mode: INDIRECT}, is8BitM)
	ret[0xD3] = NewUmbrellaRead(cmp, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xD5] = NewUmbrellaRead(cmp, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0xD7] = NewUmbrellaRead(cmp, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0xD9] = NewUmbrellaRead(cmp, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0xDD] = NewUmbrellaRead(cmp, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0xDF] = NewUmbrellaRead(cmp, &Long{mode: BASE_MODE_X}, is8BitM)

	//all 3 cpX
	ret[0xE0] = NewUmbrellaRead(cpX, &Immediate{mode: CHECK_PARENT}, is8BitX)
	ret[0xE4] = NewUmbrellaRead(cpX, &Direct{mode: BASE_MODE}, is8BitX)
	ret[0xEC] = NewUmbrellaRead(cpX, &Absolute{mode: BASE_MODE}, is8BitX)

	//all 3 cpY
	ret[0xC0] = NewUmbrellaRead(cpY, &Immediate{mode: CHECK_PARENT}, is8BitX)
	ret[0xC4] = NewUmbrellaRead(cpY, &Direct{mode: BASE_MODE}, is8BitX)
	ret[0xCC] = NewUmbrellaRead(cpY, &Absolute{mode: BASE_MODE}, is8BitX)

	//ADC x15
	ret[0x61] = NewUmbrellaRead(adc, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0x63] = NewUmbrellaRead(adc, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0x65] = NewUmbrellaRead(adc, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0x67] = NewUmbrellaRead(adc, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0x69] = NewUmbrellaRead(adc, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0x6D] = NewUmbrellaRead(adc, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0x6F] = NewUmbrellaRead(adc, &Long{mode: BASE_MODE}, is8BitM)
	ret[0x71] = NewUmbrellaRead(adc, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x72] = NewUmbrellaRead(adc, &Direct{mode: INDIRECT}, is8BitM)
	ret[0x73] = NewUmbrellaRead(adc, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0x75] = NewUmbrellaRead(adc, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0x77] = NewUmbrellaRead(adc, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0x79] = NewUmbrellaRead(adc, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0x7D] = NewUmbrellaRead(adc, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0x7F] = NewUmbrellaRead(adc, &Long{mode: BASE_MODE_X}, is8BitM)

	//SBC x15
	ret[0xE1] = NewUmbrellaRead(sbc, &Direct{mode: INDEXED_INDIRECT}, is8BitM)
	ret[0xE3] = NewUmbrellaRead(sbc, &StackS{mode: BASE_MODE}, is8BitM)
	ret[0xE5] = NewUmbrellaRead(sbc, &Direct{mode: BASE_MODE}, is8BitM)
	ret[0xE7] = NewUmbrellaRead(sbc, &Direct{mode: INDIRECT_LONG}, is8BitM)
	ret[0xE9] = NewUmbrellaRead(sbc, &Immediate{mode: CHECK_PARENT}, is8BitM)
	ret[0xED] = NewUmbrellaRead(sbc, &Absolute{mode: BASE_MODE}, is8BitM)
	ret[0xEF] = NewUmbrellaRead(sbc, &Long{mode: BASE_MODE}, is8BitM)
	ret[0xF1] = NewUmbrellaRead(sbc, &Direct{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xF2] = NewUmbrellaRead(sbc, &Direct{mode: INDIRECT}, is8BitM)
	ret[0xF3] = NewUmbrellaRead(sbc, &StackS{mode: INDIRECT_INDEXED}, is8BitM)
	ret[0xF5] = NewUmbrellaRead(sbc, &Direct{mode: BASE_MODE_X}, is8BitM)
	ret[0xF7] = NewUmbrellaRead(sbc, &Direct{mode: INDIRECT_LONG_INDEXED}, is8BitM)
	ret[0xF9] = NewUmbrellaRead(sbc, &Absolute{mode: BASE_MODE_Y}, is8BitM)
	ret[0xFD] = NewUmbrellaRead(sbc, &Absolute{mode: BASE_MODE_X}, is8BitM)
	ret[0xFF] = NewUmbrellaRead(sbc, &Long{mode: BASE_MODE_X}, is8BitM)

	//transfer to and from direct register/ accumulator
	ret[0xAA] = &TwoCycleImplied{instructionFunc: tax}
	ret[0xA8] = &TwoCycleImplied{instructionFunc: tay}
	ret[0xBA] = &TwoCycleImplied{instructionFunc: tsx}
	ret[0x8A] = &TwoCycleImplied{instructionFunc: txa}
	ret[0x9A] = &TwoCycleImplied{instructionFunc: txs}
	ret[0x9B] = &TwoCycleImplied{instructionFunc: txy}
	ret[0x98] = &TwoCycleImplied{instructionFunc: tya}
	ret[0xBB] = &TwoCycleImplied{instructionFunc: tyx}

	//transfer to and from C accumulator/ S/D
	ret[0x5B] = &TwoCycleImplied{instructionFunc: tcd}
	ret[0x1B] = &TwoCycleImplied{instructionFunc: tcs}
	ret[0x7B] = &TwoCycleImplied{instructionFunc: tdc}
	ret[0x3B] = &TwoCycleImplied{instructionFunc: tsc}

	//TODO check B-flag weirdness
	//Notes: PLA sets Z and N according to content of A. The B-flag and unused flags cannot be changed by PLP, these flags are always written as "1" by PHP.
	// stack Push/Pull implied instructions
	ret[0x2B] = &Ipld{}
	ret[0x28] = &Iplp{}
	ret[0xAB] = &Iplb{}
	ret[0x8B] = &Iphb{}
	ret[0x4B] = &Iphk{}
	ret[0x0B] = &Iphd{}
	ret[0x08] = &Iphp{}

	ret[0x48] = &PushAXY{flag: FlagM, register: func(cpu *CPU) uint16 { return cpu.r.A }}
	ret[0xDA] = &PushAXY{flag: FlagX, register: func(cpu *CPU) uint16 { return cpu.r.GetX() }}
	ret[0x5A] = &PushAXY{flag: FlagX, register: func(cpu *CPU) uint16 { return cpu.r.GetY() }}

	ret[0x68] = &PullAXY{flag: FlagM, register: func(val uint16, cpu *CPU) uint16 { return cpu.r.SetA(val) }}
	ret[0xFA] = &PullAXY{flag: FlagX, register: func(val uint16, cpu *CPU) uint16 { return cpu.r.SetX(val) }}
	ret[0x7A] = &PullAXY{flag: FlagX, register: func(val uint16, cpu *CPU) uint16 { return cpu.r.SetY(val) }}

	//PEA/PEI/PER
	ret[0xF4] = NewUmbrellaWrite(peAI, &Immediate{mode: LOCKED_16}, false, true, false, true, isNot8Bit)
	ret[0xD4] = NewUmbrellaWrite(peAI, &Direct{mode: BASE_MODE, isPEI: true}, false, true, false, true, isNot8Bit)
	ret[0x62] = NewUmbrellaWrite(per, &Immediate{mode: LOCKED_16}, false, false, false, true, isNot8Bit)

	//MVN/MVP
	ret[0x44] = &SrcDest{isPositive: true}
	ret[0x54] = &SrcDest{isPositive: false}

	return ret
}

//TODO many instructions are using address + 1 now without masking 24 bits this CAN OVERFLOW

// JMP with absolute addressing
type JMP_Abs struct {
	state    int
	lowByte  byte
	highByte byte
	address  uint16
}

func (i *JMP_Abs) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		cpu.r.PC = i.address
		return true
	}
	return false
}

func (i *JMP_Abs) Reset(cpu *CPU) {
	i.state = 0
}

// JMP with long addressing
type JMP_Long struct {
	state    int
	lowByte  byte
	highByte byte
	pbByte   byte
	address  uint16
}

func (i *JMP_Long) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.state++
	case 2:
		i.pbByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		cpu.r.PC = i.address
		cpu.r.PB = i.pbByte
		return true
	}
	return false
}

func (i *JMP_Long) Reset(cpu *CPU) {
	i.state = 0
}

// JMP with absolute(indirect) addressing
type JMP_AbsIndirect struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

func (i *JMP_AbsIndirect) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.lowByte = cpu.readByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		i.highByte = cpu.readByte(uint32(i.pointerAddress + 1))

		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *JMP_AbsIndirect) Reset(cpu *CPU) {
	i.state = 0
}

// JMP with absolute(indexed indirect) addressing
type JMP_AbsIndexedIndirect struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

func (i *JMP_AbsIndexedIndirect) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.pointerAddress += cpu.r.GetX()
		i.state++
	case 3:
		i.lowByte = cpu.readByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 4:
		i.highByte = cpu.readByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *JMP_AbsIndexedIndirect) Reset(cpu *CPU) {
	i.state = 0
}

// JMP with absolute long addressing
type JMP_AbsLong struct {
	state int

	lowByte  byte
	highByte byte
	pbByte   byte

	pointerAddress uint16
}

func (i *JMP_AbsLong) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.lowByte = cpu.readByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		i.highByte = cpu.readByte(uint32(i.pointerAddress + 1))
		i.state++
	case 4:
		i.pbByte = cpu.readByte(uint32(i.pointerAddress + 2))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		cpu.r.PB = i.pbByte
		return true
	}
	return false
}

func (i *JMP_AbsLong) Reset(cpu *CPU) {
	i.state = 0
}

// Jump to SubRoutine with absolute addressing
type JSR_Abs struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// MLB active TODO
func (i *JSR_Abs) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.highByte, i.lowByte = splitWord(cpu.r.PC - 1)
		cpu.PushByte(i.highByte)
		i.state++
	case 3:
		cpu.PushByte(i.lowByte)
		i.state++
	case 4:
		cpu.r.PC = i.pointerAddress
		return true
	}
	return false
}

func (i *JSR_Abs) Reset(cpu *CPU) {
	i.state = 0
}

// Jump to Subroutine Long
type JSL struct {
	state    int
	lowByte  byte
	highByte byte
	address  uint16
}

// the emulation test case for this so 22.e.json seems to not wrap the stack pointer
// future me: nor should it
func (i *JSL) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.highByte, i.lowByte = splitWord(cpu.r.PC)
		cpu.r.SetStack(cpu.r.S)
		cpu.PushByteNewOpCode(cpu.r.PB)
		i.state++
	case 3:
		i.state++
	case 4:
		cpu.r.PB = cpu.fetchByte()
		cpu.r.PC = i.address
		i.state++
	case 5:
		cpu.PushByteNewOpCode(i.highByte)
		i.state++
	case 6:
		cpu.PushByteNewOpCode(i.lowByte)
		cpu.r.SetStack(cpu.r.S)
		return true
	}
	return false
}

func (i *JSL) Reset(cpu *CPU) {
	i.state = 0
}

// Jump to SubRoutine with absolute(indexed indirect) addressing
// another new instruction, new pain
type JSR_AbsIndexedIndirect struct {
	state int

	lowByte  byte
	highByte byte

	lowByteS byte

	pointerAddress uint16
}

func (i *JSR_AbsIndexedIndirect) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte, i.lowByteS = splitWord(cpu.r.PC)
		cpu.r.SetStack(cpu.r.S)
		cpu.PushByteNewOpCode(i.highByte)
		i.state++
	case 2:
		cpu.PushByteNewOpCode(i.lowByteS)
		cpu.r.SetStack(cpu.r.S)
		i.state++
	case 3:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 4:
		i.pointerAddress += cpu.r.GetX()
		i.state++
	case 5:
		i.lowByte = cpu.readByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 6:
		i.highByte = cpu.readByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *JSR_AbsIndexedIndirect) Reset(cpu *CPU) {
	i.state = 0
}

// return from interrupt instruction
type RTI struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *RTI) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		if cpu.r.E {
			//TODO check B-flag weirdness Note: RTI cannot modify the B-Flag or the unused flag.
			//i.lowByte |= FlagM
			i.lowByte |= 0x30 //m and x flags are always 1 in emulation mode
		}
		cpu.r.setP(i.lowByte)
		i.state++
	case 3:
		i.lowByte = cpu.PopByte()
		i.state++
	case 4:
		i.highByte = cpu.PopByte()
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		if cpu.r.E {
			return true
		}
		i.state++
	case 5:
		cpu.r.PB = cpu.PopByte()
		return true
	}
	return false
}

func (i *RTI) Reset(cpu *CPU) {
	i.state = 0
}

// return from subroutine long instruction
// return from subroutine instruction
type RtsRtl struct {
	state int

	long bool

	lowByte  byte
	highByte byte
}

// test data is trying to read form page 2 again in emulation mode beware
// as it should!!!
func (i *RtsRtl) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		if i.long {
			cpu.r.SetStack(cpu.r.S)
			i.lowByte = cpu.PopByteNewOpCode()
		} else {
			i.lowByte = cpu.PopByte()
		}
		i.state++
	case 3:
		if i.long {
			i.highByte = cpu.PopByteNewOpCode()
		} else {
			i.highByte = cpu.PopByte()
		}
		cpu.r.PC = createWord(i.highByte, i.lowByte) + 1
		i.state++
	case 4:
		if i.long {
			cpu.r.PB = cpu.PopByteNewOpCode()
			cpu.r.SetStack(cpu.r.S)
		}
		return true
	}
	return false
}

func (i *RtsRtl) Reset(cpu *CPU) {
	i.state = 0
}

// BRL represents the BRL or branch always long instruction
type BRL struct {
	state int

	offsetL byte
	offsetH byte
}

func (i *BRL) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offsetL = cpu.fetchByte()
		i.state++
	case 1:
		i.offsetH = cpu.fetchByte()
		i.state++
	case 2:
		cpu.r.PC += rel16(createWord(i.offsetH, i.offsetL))
		return true
	}
	return false
}

func (i *BRL) Reset(cpu *CPU) {
	i.state = 0
}

// all one bit branch instructions
// BCC BCS BEQ BMI BNE BPL BRA BVC BVS
type OneByteBranch struct {
	state int

	pcTmp  uint16
	offset uint8

	shouldBranch func(cpu *CPU) bool
}

func (i *OneByteBranch) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offset = cpu.fetchByte()
		if !i.shouldBranch(cpu) {
			return true
		}
		i.state++
	case 1:
		i.pcTmp = cpu.r.PC
		rel8(cpu, i.offset)
		if cpu.r.E && isPageBoundaryCrossed(i.pcTmp, cpu.r.PC) {
			i.state++
		} else {
			return true
		}
	case 2:
		return true
	}
	return false
}

func (i *OneByteBranch) Reset(cpu *CPU) {
	i.state = 0
}

type softwareInterrupt struct {
	state int

	lowByte  byte
	highByte byte

	eAddress uint32
	nAddress uint32

	address uint32
}

func (i *softwareInterrupt) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		//discard the next byte and increase PC
		cpu.fetchByte()
		i.state++
		if cpu.r.E {
			i.state++
		}
	case 1:
		cpu.PushByte(cpu.r.PB)
		i.state++
	case 2:
		i.highByte, i.lowByte = splitWord(cpu.r.PC)
		cpu.PushByte(i.highByte)
		i.state++
	case 3:
		cpu.PushByte(i.lowByte)
		i.state++
	case 4:
		if cpu.r.E {
			cpu.PushByte(cpu.r.P | FlagX)
		} else {
			cpu.PushByte(cpu.r.P)
		}
		i.state++
	case 5:
		if cpu.r.E {
			i.address = i.eAddress
		} else {
			i.address = i.nAddress
		}

		i.lowByte = cpu.readByte(i.address)
		i.state++
	case 6:
		i.highByte = cpu.readByte(i.address + 1)

		cpu.r.PB = 0x00
		cpu.r.PC = createWord(i.highByte, i.lowByte)

		cpu.r.setFlag(FlagD, true)
		cpu.r.setFlag(FlagI, false)
		return true
	}
	return false
}

func (i *softwareInterrupt) Reset(cpu *CPU) {
	i.state = 0
}

// has to be called before the next opcode is fetched to have accurate PC
type NmiIrqSequence struct {
	state int

	lowByte  byte
	highByte byte

	eAddress uint32
	nAddress uint32

	address uint32
}

func (i *NmiIrqSequence) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		cpu.PushByte(cpu.r.PB)
		i.state++
	case 1:
		i.highByte, i.lowByte = splitWord(cpu.r.PC)
		cpu.PushByte(i.highByte)
		i.state++
	case 2:
		cpu.PushByte(i.lowByte)
		i.state++
	case 3:
		if cpu.r.E {
			cpu.PushByte(cpu.r.P & ^FlagX)
		} else {
			cpu.PushByte(cpu.r.P)
		}
		i.state++
	case 4:
		if cpu.r.E {
			i.address = i.eAddress
		} else {
			i.address = i.nAddress
		}

		i.lowByte = cpu.readByte(i.address)
		i.state++
	case 5:
		i.highByte = cpu.readByte(i.address + 1)

		cpu.r.PB = 0x00
		cpu.r.PC = createWord(i.highByte, i.lowByte)

		cpu.r.setFlag(FlagD, true)
		cpu.r.setFlag(FlagI, false)
		return true
	}
	return false
}

func (i *NmiIrqSequence) Reset(cpu *CPU) {
	if cpu.r.E {
		i.state = 1
	} else {
		i.state = 0
	}
}

type AbortSequence struct {
	state int

	lowByte  byte
	highByte byte

	eAddress uint32
	nAddress uint32

	address uint32

	PC uint16
}

func (i *AbortSequence) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		cpu.PushByte(cpu.r.PB)
		i.state++
	case 1:
		i.highByte, i.lowByte = splitWord(i.PC)
		cpu.PushByte(i.highByte)
		i.state++
	case 2:
		cpu.PushByte(i.lowByte)
		i.state++
	case 3:
		cpu.PushByte(cpu.r.P)
		i.state++
	case 4:
		if cpu.r.E {
			i.address = i.eAddress
		} else {
			i.address = i.nAddress
		}

		i.lowByte = cpu.readByte(i.address)
		i.state++
	case 5:
		i.highByte = cpu.readByte(i.address + 1)

		cpu.r.PB = 0x00
		cpu.r.PC = createWord(i.highByte, i.lowByte)

		cpu.r.setFlag(FlagD, true)
		cpu.r.setFlag(FlagI, false)
		return true
	}
	return false
}

func (i *AbortSequence) Reset(cpu *CPU) {
	if cpu.currentInstruction != nil {
		i.PC = cpu.r.instrPC
	} else {
		i.PC = cpu.r.PC
	}

	if cpu.r.E {
		i.state = 1
	} else {
		i.state = 0
	}
}

type ResetSequence struct {
	state int

	lowByte  byte
	highByte byte

	eAddress uint32

	PC uint16
}

// TODO unsure if this is correct or not but i heard reset takes 6-9 instructions and this with the signal catch is 8
func (i *ResetSequence) Step(cpu *CPU) bool {
	switch i.state {
	case 0, 1, 2, 3, 4:
		i.state++
	case 5:
		i.lowByte = cpu.readByte(i.eAddress)
		i.state++
	case 6:
		i.highByte = cpu.readByte(i.eAddress + 1)

		cpu.r.E = true

		cpu.r.PB = 0x00
		cpu.r.DB = 0x00
		cpu.r.D = 0x0000

		cpu.r.S = 0x01FF

		// set M X and I to 1
		cpu.r.setP(0x34)

		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *ResetSequence) Reset(cpu *CPU) {
	i.state = 0
}

// REP SEP
type RepSep struct {
	state int

	reset   bool
	operand byte
}

func (i *RepSep) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.operand = cpu.fetchByte()
		i.state++
	case 1:
		newP := cpu.r.P
		cpu.previousIFlag = int(newP & FlagI)

		if i.reset {
			newP &= ^i.operand
		} else {
			newP |= i.operand
		}
		if cpu.r.E {
			newP |= 0x30
		}
		cpu.r.setP(newP)
		return true
	}
	return false
}

func (i *RepSep) Reset(cpu *CPU) {
	i.state = 0
}

type StpWai struct {
	state int

	executionState ExecutionState
}

func (i *StpWai) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		cpu.executionState = i.executionState
		return true
	}
	return false
}

func (i *StpWai) Reset(cpu *CPU) {
	i.state = 0
}

type XBA struct {
	state int

	lowByte, highByte byte
}

func (i *XBA) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.highByte, i.lowByte = splitWord(cpu.r.A)
		i.state++
	case 1:
		cpu.r.A = (createWord(i.lowByte, i.highByte))
		cpu.r.setFlag(FlagN, i.highByte&(0x80) == 0)
		cpu.r.setFlag(FlagZ, i.highByte != 0)
		return true
	}
	return false
}

func (i *XBA) Reset(cpu *CPU) {
	i.state = 0
}
