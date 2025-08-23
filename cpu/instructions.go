package cpu

// Instruction represents a single CPU instruction, executed one cycle at a time.
type Instruction interface {
	// Step performs one cycle of the instruction's execution.
	// It returns true if the instruction is complete, false otherwise.
	Step(cpu *CPU) bool
	Reset(cpu *CPU)
}

func NewHWInterruptMap() map[int]Instruction {
	ret := make(map[int]Instruction)
	ret[irqId] = &NmiIrqSequence{eAddress: 0x00FFFE, nAddress: 0x00FFEE}
	ret[nmiId] = &NmiIrqSequence{eAddress: 0x00FFFA, nAddress: 0x00FFEA}
	ret[abortId] = &AbortSequence{eAddress: 0x00FFF8, nAddress: 0x00FFE8}
	ret[resetId] = &ResetSequence{eAddress: 0x00FFFC}

	return ret
}

func NewInstructionMap() map[byte]Instruction {
	ret := make(map[byte]Instruction)

	ret[0x4C] = &I4C{}
	ret[0x5C] = &I5C{}
	ret[0x6C] = &I6C{}
	ret[0x7C] = &I7C{}
	ret[0xDC] = &IDC{}
	ret[0xFC] = &IFC{}
	ret[0x20] = &I20{}
	ret[0x22] = &I22{}

	ret[0x40] = &I40{}
	ret[0x6B] = &I6B{}
	ret[0x60] = &I60{}

	ret[0x82] = &I82{}

	// I80 represents the BRA or branch always instruction
	ret[0x80] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return true }}
	// I10 represents the BPL or branch if positive instruction
	ret[0x10] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagN) }}
	// I30 represents the BMI or branch if not positive instruction
	ret[0x30] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagN) }}
	// I90 represents the BCC or branch if no carry
	ret[0x90] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagC) }}
	// IB0 represents the BCS or branch if carry instruction
	ret[0xB0] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagC) }}
	// IF0 represents the BEQ or branch if zero instruction
	ret[0xF0] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagZ) }}
	// ID0 represents the BNE or branch if not zero instruction
	ret[0xD0] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagZ) }}
	// I50 represents the BVC or branch if not overflow instruction
	ret[0x50] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return !cpu.r.hasFlag(FlagV) }}
	// I70 represents the BVS or branch if overflow instruction
	ret[0x70] = &OneBitBranch{shouldBranch: func(cpu *CPU) bool { return cpu.r.hasFlag(FlagV) }}

	// I00 represents the BRK or break (software interrupt) instruction
	ret[0x00] = &softwareInterrupt{eAddress: 0x00FFFE, nAddress: 0x00FFE6}
	// I02 represents the COP (software interrupt) instruction
	ret[0x02] = &softwareInterrupt{eAddress: 0x00FFF4, nAddress: 0x00FFE4}

	ret[0x18] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagC, true) }}
	ret[0xD8] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagD, true) }}
	ret[0x58] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagI, true) }}
	ret[0xB8] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagV, true) }}
	ret[0x38] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagC, false) }}
	ret[0xF8] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagD, false) }}
	ret[0x78] = &CDIVflagSetter{changeFlag: func(cpu *CPU) { cpu.r.setFlag(FlagI, false) }}

	//rep
	ret[0xC2] = &RepSep{reset: true}
	//sep
	ret[0xE2] = &RepSep{reset: false}

	ret[0xFB] = &IFB{}

	//STP/WAI
	ret[0xDB] = &StpWai{executionState: stopState}
	ret[0xCB] = &StpWai{executionState: waitState}

	ret[0xEB] = &IEB{}

	//the NOP instructions
	ret[0xEA] = &IEA{}
	ret[0x42] = &I42{}

	//the shift and rotate instructions
	ret[0x0A] = &Accumulator{instructionFunc: asl}
	ret[0x4A] = &Accumulator{instructionFunc: lsr}
	ret[0x6A] = &Accumulator{instructionFunc: ror}
	ret[0x2A] = &Accumulator{instructionFunc: rol}

	ret[0x46] = &Umbrella{instructionFunc: lsr, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x26] = &Umbrella{instructionFunc: rol, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x06] = &Umbrella{instructionFunc: asl, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x66] = &Umbrella{instructionFunc: ror, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}

	ret[0x56] = &Umbrella{instructionFunc: lsr, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x36] = &Umbrella{instructionFunc: rol, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x16] = &Umbrella{instructionFunc: asl, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x76] = &Umbrella{instructionFunc: ror, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}

	ret[0x4E] = &Umbrella{instructionFunc: lsr, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x2E] = &Umbrella{instructionFunc: rol, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x0E] = &Umbrella{instructionFunc: asl, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x6E] = &Umbrella{instructionFunc: ror, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}

	ret[0x5E] = &Umbrella{instructionFunc: lsr, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0x3E] = &Umbrella{instructionFunc: rol, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0x1E] = &Umbrella{instructionFunc: asl, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0x7E] = &Umbrella{instructionFunc: ror, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}

	//Test and Set/Test and Reset bits
	ret[0x1C] = &Umbrella{instructionFunc: trb, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x0C] = &Umbrella{instructionFunc: tsb, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x14] = &Umbrella{instructionFunc: trb, mode: WRITE_RAM, checkM: true, addressMode: &Direct{isPEI: false, mode: BASE_MODE}}
	ret[0x04] = &Umbrella{instructionFunc: tsb, mode: WRITE_RAM, checkM: true, addressMode: &Direct{isPEI: false, mode: BASE_MODE}}

	//all 14 sta instructions
	ret[0x81] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, reverseWrites: true, combineExecuteAndWrite: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0x83] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0x85] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x87] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, reverseWrites: true, checkM: true, combineExecuteAndWrite: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0x8D] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x8F] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0x91] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, reverseWrites: true, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED}}
	ret[0x92] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, reverseWrites: true, checkM: true, combineExecuteAndWrite: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0x93] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0x95] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x97] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, reverseWrites: true, checkM: true, combineExecuteAndWrite: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0x99] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE_Y}}
	ret[0x9D] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0x9F] = &Umbrella{instructionFunc: sta, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Long{mode: BASE_MODE_X}}

	//all 3 stx instructions
	ret[0x86] = &Umbrella{instructionFunc: stx, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x8E] = &Umbrella{instructionFunc: stx, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x96] = &Umbrella{instructionFunc: stx, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE_Y}}

	//all 3 sty instructions
	ret[0x84] = &Umbrella{instructionFunc: sty, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x8C] = &Umbrella{instructionFunc: sty, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x94] = &Umbrella{instructionFunc: sty, mode: WRITE_RAM, checkX: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE_X}}

	//all 4 stz instructions
	ret[0x64] = &Umbrella{instructionFunc: stz, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x74] = &Umbrella{instructionFunc: stz, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x9C] = &Umbrella{instructionFunc: stz, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x9E] = &Umbrella{instructionFunc: stz, mode: WRITE_RAM, checkM: true, executeInFetch: true, combineExecuteAndWrite: true, reverseWrites: true, addressMode: &Absolute{mode: BASE_MODE_X}}

	//all 15 LDA instructions
	ret[0xA1] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0xA3] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0xA5] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xA7] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0xA9] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xAD] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xAF] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0xB1] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0xB2] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0xB3] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0xB5] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xB7] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0xB9] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0xBD] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0xBF] = &Umbrella{instructionFunc: lda, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//all 5 LDX instructions
	ret[0xA2] = &Umbrella{instructionFunc: ldx, mode: READ_RAM, checkX: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xA6] = &Umbrella{instructionFunc: ldx, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xAE] = &Umbrella{instructionFunc: ldx, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xB6] = &Umbrella{instructionFunc: ldx, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE_Y}}
	ret[0xBE] = &Umbrella{instructionFunc: ldx, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}

	//all 5 LDY instructions
	ret[0xA0] = &Umbrella{instructionFunc: ldy, mode: READ_RAM, checkX: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xA4] = &Umbrella{instructionFunc: ldy, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xAC] = &Umbrella{instructionFunc: ldy, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xB4] = &Umbrella{instructionFunc: ldy, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xBC] = &Umbrella{instructionFunc: ldy, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}

	//test BITs
	ret[0x89] = &Umbrella{instructionFunc: bit_imm, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0x24] = &Umbrella{instructionFunc: bit, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x2C] = &Umbrella{instructionFunc: bit, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x34] = &Umbrella{instructionFunc: bit, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x3C] = &Umbrella{instructionFunc: bit, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}

	//bitwise AND all 15
	ret[0x21] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0x23] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0x25] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x27] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0x29] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0x2D] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x2F] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0x31] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0x32] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0x33] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0x35] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x37] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0x39] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0x3D] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0x3F] = &Umbrella{instructionFunc: and, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//bitwise EOR all 15
	ret[0x41] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0x43] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0x45] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x47] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0x49] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0x4D] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x4F] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0x51] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0x52] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0x53] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0x55] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x57] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0x59] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0x5D] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0x5F] = &Umbrella{instructionFunc: eor, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//bitwise ORA all 15
	ret[0x01] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0x03] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0x05] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x07] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0x09] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0x0D] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x0F] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0x11] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0x12] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0x13] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0x15] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x17] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0x19] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0x1D] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0x1F] = &Umbrella{instructionFunc: ora, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//DECrement
	ret[0x3A] = &Accumulator{instructionFunc: dec}
	ret[0xC6] = &Umbrella{instructionFunc: dec, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xCE] = &Umbrella{instructionFunc: dec, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xD6] = &Umbrella{instructionFunc: dec, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xDE] = &Umbrella{instructionFunc: dec, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0xCA] = &TwoCycleImplied{instructionFunc: decX}
	ret[0x88] = &TwoCycleImplied{instructionFunc: decY}

	//INCrement
	ret[0x1A] = &Accumulator{instructionFunc: inc}
	ret[0xE6] = &Umbrella{instructionFunc: inc, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xEE] = &Umbrella{instructionFunc: inc, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xF6] = &Umbrella{instructionFunc: inc, mode: WRITE_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xFE] = &Umbrella{instructionFunc: inc, mode: WRITE_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X}}
	ret[0xE8] = &TwoCycleImplied{instructionFunc: incX}
	ret[0xC8] = &TwoCycleImplied{instructionFunc: incY}

	//175 instructions above this comment +-1 depending if i can count or not

	//CoMPare all 15
	ret[0xC1] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0xC3] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0xC5] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xC7] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0xC9] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xCD] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xCF] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0xD1] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0xD2] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0xD3] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0xD5] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xD7] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0xD9] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0xDD] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0xDF] = &Umbrella{instructionFunc: cmp, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//all 3 cpX
	ret[0xE0] = &Umbrella{instructionFunc: cpX, mode: READ_RAM, checkX: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xE4] = &Umbrella{instructionFunc: cpX, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xEC] = &Umbrella{instructionFunc: cpX, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE}}

	//all 3 cpY
	ret[0xC0] = &Umbrella{instructionFunc: cpY, mode: READ_RAM, checkX: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xC4] = &Umbrella{instructionFunc: cpY, mode: READ_RAM, checkX: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xCC] = &Umbrella{instructionFunc: cpY, mode: READ_RAM, checkX: true, addressMode: &Absolute{mode: BASE_MODE}}

	//ADC x15
	ret[0x61] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0x63] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0x65] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0x67] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0x69] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0x6D] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0x6F] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0x71] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0x72] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0x73] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0x75] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0x77] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0x79] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0x7D] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0x7F] = &Umbrella{instructionFunc: adc, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	//ADC x15
	ret[0xE1] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDEXED_INDIRECT}}
	ret[0xE3] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: BASE_MODE}}
	ret[0xE5] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE}}
	ret[0xE7] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG}}
	ret[0xE9] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Immediate{mode: CHECK_PARENT}}
	ret[0xED] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE}}
	ret[0xEF] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE}}
	ret[0xF1] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_INDEXED, checkP: true}}
	ret[0xF2] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT}}
	ret[0xF3] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &StackS{mode: INDIRECT_INDEXED}}
	ret[0xF5] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: BASE_MODE_X}}
	ret[0xF7] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Direct{mode: INDIRECT_LONG_INDEXED}}
	ret[0xF9] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_Y, checkP: true}}
	ret[0xFD] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Absolute{mode: BASE_MODE_X, checkP: true}}
	ret[0xFF] = &Umbrella{instructionFunc: sbc, mode: READ_RAM, checkM: true, addressMode: &Long{mode: BASE_MODE_X}}

	ret[0xAA] = &TwoCycleImplied{instructionFunc: tax}
	ret[0xA8] = &TwoCycleImplied{instructionFunc: tay}
	ret[0xBA] = &TwoCycleImplied{instructionFunc: tsx}
	ret[0x8A] = &TwoCycleImplied{instructionFunc: txa}
	ret[0x9A] = &TwoCycleImplied{instructionFunc: txs}
	ret[0x9B] = &TwoCycleImplied{instructionFunc: txy}
	ret[0x98] = &TwoCycleImplied{instructionFunc: tya}
	ret[0xBB] = &TwoCycleImplied{instructionFunc: tyx}
	return ret
}

//TODO many instructions are using address + 1 now without masking 24 bits this CAN OVERFLOW

// I4C represents the JMP $XXXX instruction (opcode 0x4C)
type I4C struct {
	state    int
	lowByte  byte
	highByte byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
func (i *I4C) Step(cpu *CPU) bool {
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

func (i *I4C) Reset(cpu *CPU) {
	i.state = 0
}

// I5C represents the JMP $XXXXXX instruction (opcode 0x5C)
type I5C struct {
	state    int
	lowByte  byte
	highByte byte
	pbByte   byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
func (i *I5C) Step(cpu *CPU) bool {
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

func (i *I5C) Reset(cpu *CPU) {
	i.state = 0
}

// I6C represents the JMP [nnnn] instruction (opcode 0x6C)
type I6C struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *I6C) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		//this read doesnt have to be mapped because it defaults to bank 0x00
		i.lowByte = cpu.bus.ReadByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		var highByteAddress uint32
		/* the tests are passing without the glitch so its removed for now
		if i.pointerAddress&0x00FF == 0x00FF {
			// The glitch! The high byte is fetched from the start of the same page.
			highByteAddress = uint32(i.pointerAddress & 0xFF00)
		} else {
		*/
		highByteAddress = uint32(i.pointerAddress + 1)
		//}
		i.highByte = cpu.bus.ReadByte(highByteAddress)

		finalAddress := createWord(i.highByte, i.lowByte)
		cpu.r.PC = finalAddress

		return true
	}
	return false
}

func (i *I6C) Reset(cpu *CPU) {
	i.state = 0
}

// I7C represents the JMP [nnnn+X] instruction (opcode 0x7C)
type I7C struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *I7C) Step(cpu *CPU) bool {
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
		i.lowByte = cpu.bus.ReadByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 4:
		i.highByte = cpu.bus.ReadByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *I7C) Reset(cpu *CPU) {
	i.state = 0
}

// IDC represents the JMP FAR[nnnn] instruction
type IDC struct {
	state int

	lowByte  byte
	highByte byte
	pbByte   byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *IDC) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.lowByte = cpu.bus.ReadByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		i.highByte = cpu.bus.ReadByte(uint32(i.pointerAddress + 1))
		i.state++
	case 4:
		i.pbByte = cpu.bus.ReadByte(uint32(i.pointerAddress + 2))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		cpu.r.PB = i.pbByte
		return true
	}
	return false
}

func (i *IDC) Reset(cpu *CPU) {
	i.state = 0
}

// I20 represents the CALL nnnn instruction
type I20 struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
// MLB active TODO
func (i *I20) Step(cpu *CPU) bool {
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

func (i *I20) Reset(cpu *CPU) {
	i.state = 0
}

// I22 represents the CALL nnnnnn instruction
type I22 struct {
	state    int
	lowByte  byte
	highByte byte
	pbByte   byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
// the emulation test case for this so 22.e.json seems to not wrap the stack pointer
// it also has a faulty test case on top of that. maybe im wrong but additional debug needed later TODO
// and im not even sure this is cycle accurate.
func (i *I22) Step(cpu *CPU) bool {
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
		cpu.PushByte(cpu.r.PB)
		i.state++
	case 3:
		i.state++
	case 4:
		i.pbByte = cpu.fetchByte()
		cpu.r.PC = i.address
		cpu.r.PB = i.pbByte
		i.state++
	case 5:
		cpu.PushByte(i.highByte)
		i.state++
	case 6:
		cpu.PushByte(i.lowByte)
		return true
	}
	return false
}

func (i *I22) Reset(cpu *CPU) {
	i.state = 0
}

// IFC represents the CALL [nnnn + X] instruction
type IFC struct {
	state int

	lowByte  byte
	highByte byte

	lowByteS byte

	pointerAddress uint16
}

func (i *IFC) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte, i.lowByteS = splitWord(cpu.r.PC)
		cpu.PushByte(i.highByte)
		i.state++
	case 2:
		cpu.PushByte(i.lowByteS)
		i.state++
	case 3:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 4:
		i.pointerAddress += cpu.r.GetX()
		i.state++
	case 5:
		i.lowByte = cpu.bus.ReadByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 6:
		i.highByte = cpu.bus.ReadByte(mapOffsetToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *IFC) Reset(cpu *CPU) {
	i.state = 0
}

// I40 represents the RTI or return from interrupt instruction
type I40 struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I40) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		if cpu.r.E {
			i.lowByte |= 0x30 //m and x flags are always 1 in emulation mode
		}
		cpu.r.P = i.lowByte
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

func (i *I40) Reset(cpu *CPU) {
	i.state = 0
}

// I6B represents the RTL or return from subroutine long instruction
type I6B struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I6B) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		i.state++
	case 3:
		i.highByte = cpu.PopByte()
		cpu.r.PC = createWord(i.highByte, i.lowByte) + 1
		i.state++
	case 4:
		cpu.r.PB = cpu.PopByte()
		return true
	}
	return false
}

func (i *I6B) Reset(cpu *CPU) {
	i.state = 0
}

// I60 represents the RTS or return from subroutine instruction
type I60 struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I60) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		i.state++
	case 3:
		i.highByte = cpu.PopByte()
		i.state++
	case 4:
		cpu.r.PC = createWord(i.highByte, i.lowByte) + 1
		return true
	}
	return false
}

func (i *I60) Reset(cpu *CPU) {
	i.state = 0
}

// I82 represents the BRL or branch always long instruction
type I82 struct {
	state int

	offsetL byte
	offsetH byte
}

func (i *I82) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offsetL = cpu.fetchByte()
		i.state++
	case 1:
		i.offsetH = cpu.fetchByte()
		i.state++
	case 2:
		rel16(cpu, i.offsetH, i.offsetL)
		return true
	}
	return false
}

func (i *I82) Reset(cpu *CPU) {
	i.state = 0
}

// all one bit branch instructions
// BCC BCS BEQ BMI BNE BPL BRA BVC BVS
type OneBitBranch struct {
	state int

	pcTmp  uint16
	offset uint8

	shouldBranch func(cpu *CPU) bool
}

func (i *OneBitBranch) Step(cpu *CPU) bool {
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

func (i *OneBitBranch) Reset(cpu *CPU) {
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

		i.lowByte = cpu.bus.ReadByte(i.address)
		i.state++
	case 6:
		i.highByte = cpu.bus.ReadByte(i.address + 1)

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

		i.lowByte = cpu.bus.ReadByte(i.address)
		i.state++
	case 5:
		i.highByte = cpu.bus.ReadByte(i.address + 1)

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

		i.lowByte = cpu.bus.ReadByte(i.address)
		i.state++
	case 5:
		i.highByte = cpu.bus.ReadByte(i.address + 1)

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
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.state++
	case 3:
		i.state++
	case 4:
		i.state++
	case 5:
		i.lowByte = cpu.bus.ReadByte(i.eAddress)
	case 6:
		i.highByte = cpu.bus.ReadByte(i.eAddress + 1)

		cpu.r.E = true

		cpu.r.PB = 0x00
		cpu.r.DB = 0x00
		cpu.r.D = 0x0000

		cpu.r.S = 0x01FF

		// set M X and I to 1
		cpu.r.P = 0x34

		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *ResetSequence) Reset(cpu *CPU) {
	i.state = 0
}

// CLC CLD CLI CLV SEC SED SEI
type CDIVflagSetter struct {
	state int

	changeFlag func(cpu *CPU)
}

func (i *CDIVflagSetter) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.changeFlag(cpu)
		return true
	}
	return false
}

func (i *CDIVflagSetter) Reset(cpu *CPU) {
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
		if i.reset {
			cpu.r.P &= ^i.operand
		} else {
			cpu.r.P |= i.operand
		}
		if cpu.r.E {
			cpu.r.P |= 0x30
		}
		return true
	}
	return false
}

func (i *RepSep) Reset(cpu *CPU) {
	i.state = 0
}

// the XCE or eXchange Carry and Emulation instruction
// the only instruction that can swap modes
type IFB struct {
	state int
}

func (i *IFB) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		tmp := cpu.r.hasFlag(FlagC)
		cpu.r.setFlag(FlagC, !cpu.r.E)
		cpu.r.E = tmp
		if tmp {
			cpu.r.P |= 0x30
			cpu.r.X = maskHighByte(cpu.r.X)
			cpu.r.Y = maskHighByte(cpu.r.Y)
			cpu.r.S = 0x0100 | maskHighByte(cpu.r.S)
		}
		return true
	}
	return false
}

func (i *IFB) Reset(cpu *CPU) {
	i.state = 0
}

type StpWai struct {
	state int

	executionState int
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

type IEB struct {
	state int

	lowByte, highByte byte
}

func (i *IEB) Step(cpu *CPU) bool {
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

func (i *IEB) Reset(cpu *CPU) {
	i.state = 0
}

// the NOP instruction
type IEA struct {
	state int
}

func (i *IEA) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		return true
	}
	return false
}

func (i *IEA) Reset(cpu *CPU) {
	i.state = 0
}

// the WDM or otherwise known as the 2 byte NOP
type I42 struct {
	state int
}

func (i *I42) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		cpu.fetchByte()
		return true
	}
	return false
}

func (i *I42) Reset(cpu *CPU) {
	i.state = 0
}
