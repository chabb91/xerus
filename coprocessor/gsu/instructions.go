package gsu

import (
	"fmt"
)

func (gsu *GSU) processByte() {
	immediateNum := gsu.r.getImmediateNum()
	if immediateNum != 0 {
		immediateNum--
		gsu.immediateBytes[immediateNum] = gsu.fetchedByte
		gsu.r.setImmediateNum(immediateNum)
		if immediateNum == 0 {
			gsu.immediateInstruction(gsu)
		}
		return
	} else {
		opcode := gsu.fetchedByte
		opcodeHn := opcode & 0xF0
		opcodeLn := opcode & 0x0F
		switch {
		case opcode-5 <= 0xA: //BRANCH instructions 0x05-0x0F UNTESTED
			gsu.r.setImmediateNum(1)
			gsu.immediateInstruction = branchFunc
			gsu.immediateOpcode = opcode
		case opcodeHn == 0xF0: //IWT instructions
			gsu.r.setImmediateNum(2)
			gsu.immediateInstruction = iwtFunc
			gsu.immediateOpcode = opcode
		case opcodeHn == 0xA0: //IBT instructions
			gsu.r.setImmediateNum(1)
			gsu.immediateInstruction = ibtFunc
			gsu.immediateOpcode = opcode
		case opcode-0x30 <= 0xB: //STW instructions
			gsu.ramWordStore(gsu.r.cpuRegisters[opcodeLn], gsu.sReg, gsu.r.getAltNum() == FlagAlt1, false)
			gsu.clearPrefixes()
		case opcode-0x40 <= 0xB: //LDW instructions
			gsu.ramWordLoad(gsu.r.cpuRegisters[opcodeLn], gsu.dReg, gsu.r.getAltNum() == FlagAlt1)
			gsu.clearPrefixes()
		case opcode == 0x90:
			gsu.ramWordStore(0, gsu.sReg, false, true)
			gsu.clearPrefixes()
		case opcode == 0xEF: //GET(load byte from rom)
			byte, _ := gsu.Read8(gsu.r.ROMBR, gsu.r.cpuRegisters[14])
			val := &gsu.r.cpuRegisters[gsu.dReg]
			switch gsu.r.getAltNum() {
			case FlagAlt1:
				*val &= 0x00FF
				*val |= uint16(byte) << 8
			case FlagAlt2:
				*val &= 0xFF00
				*val |= uint16(byte)
			case FlagAlt3:
				*val = uint16(int16(int8(byte))) //technically the inner int16 cast isnt needed, but cant test
			default:
				*val = uint16(byte)
			}
			gsu.clearPrefixes()
		case opcode == 0xDF: //GETC pretending as RAMB/ROMB
			switch gsu.r.getAltNum() {
			case FlagAlt2:
				gsu.r.RAMBR = byte(gsu.r.cpuRegisters[gsu.sReg]) & 1
			case FlagAlt3:
				gsu.r.ROMBR = byte(gsu.r.cpuRegisters[gsu.sReg])
			default:
				gsu.r.COLR, _ = gsu.Read8(gsu.r.ROMBR, gsu.r.cpuRegisters[14])
			}
			gsu.clearPrefixes()
		case opcode == 0x4E: //COLOR/CMODE
			if gsu.r.getAltNum() == FlagAlt1 {
				gsu.r.POR = byte(gsu.r.cpuRegisters[gsu.sReg]) & 0x1F
			} else {
				gsu.r.COLR = byte(gsu.r.cpuRegisters[gsu.sReg])
			}
			gsu.clearPrefixes()
		case opcodeHn == 0x50: //ADD/ADC instructions
			reg := opcodeLn
			signA := uint16(gsu.r.cpuRegisters[gsu.sReg])
			result32 := uint32(signA)
			signA &= 0x8000

			if gsu.r.SFR&FlagAlt1 != 0 { //adc
				result32 += uint32(min(gsu.r.SFR&FlagC, 1))
			}
			signB := uint16(0)
			if gsu.r.SFR&FlagAlt2 != 0 {
				result32 += uint32(reg)
			} else {
				signB = gsu.r.cpuRegisters[reg]
				result32 += uint32(signB)
				signB &= 0x8000
			}
			result := uint16(result32)
			gsu.r.setFlag(FlagC, result32>>16 > 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.setFlag(FlagV, result&0x8000 != signA && signA == signB)
			gsu.r.cpuRegisters[gsu.dReg] = result
			gsu.clearPrefixes()
		case opcode == 0x3D: //ALT1
			gsu.r.SFR |= FlagAlt1
			fmt.Println("SETTING ALT1")
		case opcode == 0x3E: //ALT2
			gsu.r.SFR |= FlagAlt2
			fmt.Println("SETTING ALT2")
		case opcode == 0x3F: //ALT3
			gsu.r.SFR |= (FlagAlt1 | FlagAlt2)
			fmt.Println("SETTING ALT3")
		case opcodeHn == 0x10: //TO
			dReg := opcodeLn
			if gsu.r.SFR&FlagB != 0 { //MOVE
				gsu.r.cpuRegisters[dReg] = gsu.r.cpuRegisters[gsu.sReg]
				gsu.clearPrefixes()
			} else {
				gsu.dReg = dReg
			}
		case opcodeHn == 0xB0: //FROM
			sReg := opcodeLn
			if gsu.r.SFR&FlagB != 0 { //MOVES
				val := gsu.r.cpuRegisters[sReg]
				gsu.r.cpuRegisters[gsu.dReg] = val
				gsu.r.setFlag(FlagZ, val == 0)
				gsu.r.setFlag(FlagS, val&0x8000 != 0)
				gsu.r.setFlag(FlagV, val&0x80 != 0)
				gsu.clearPrefixes()
			} else {
				gsu.sReg = sReg
			}
		case opcodeHn == 0x20: //WITH
			gsu.r.SFR |= FlagB

			reg := opcodeLn
			gsu.sReg, gsu.dReg = reg, reg
			fmt.Println("(WITH)SETTING Rd & Rs to :", reg)
		case opcode == 0x00: //STOP
			fmt.Println("STOPPING")
			gsu.r.SFR &= ^FlagGo
			gsu.r.SFR |= FlagIrq
			gsu.clearPrefixes()
		case opcode == 0x01: //NOP
			gsu.clearPrefixes()
		default:
			panic(fmt.Sprintf("GSU: unknown opcode: $%02x", opcode))
		}
	}
}

func (gsu *GSU) clearPrefixes() {
	gsu.r.SFR &= ^(FlagB | FlagAlt1 | FlagAlt2)
	gsu.sReg, gsu.dReg = 0, 0
}

func iwtFunc(gsu *GSU) {
	reg := gsu.immediateOpcode & 0xF
	hilo := uint16(gsu.immediateBytes[0])<<8 | uint16(gsu.immediateBytes[1])
	switch gsu.r.getAltNum() {
	case FlagAlt1:
		gsu.ramWordLoad(hilo, reg, false)
	case FlagAlt2:
		gsu.ramWordStore(hilo, reg, false, false)
	default:
		gsu.r.cpuRegisters[reg] = hilo
		fmt.Println("REG: ", reg, " :", gsu.r.cpuRegisters[reg])
	}
	gsu.clearPrefixes()
}

func ibtFunc(gsu *GSU) {
	reg := gsu.immediateOpcode & 0xF
	kk := gsu.immediateBytes[0]
	switch gsu.r.getAltNum() {
	case FlagAlt1:
		gsu.ramWordLoad(uint16(kk)<<1, reg, false)
	case FlagAlt2:
		gsu.ramWordStore(uint16(kk)<<1, reg, false, false)
	default:
		gsu.r.cpuRegisters[reg] = uint16(int8(kk))
		fmt.Println("IBT normal mode")
	}
	gsu.clearPrefixes()
}

// TODO UNTESTED HELPER FUNCTION
func (gsu *GSU) ramWordLoad(addr uint16, register byte, isByte bool) {
	bank := SRAM_BASE_BANK + gsu.r.RAMBR
	gsu.prevRamAddr = uint32(bank)<<16 | uint32(addr)

	lo, _ := gsu.Read8(bank, addr)
	hi := byte(0)
	if !isByte {
		hi, _ = gsu.Read8(bank, addr^1)
	}
	gsu.r.cpuRegisters[register] = uint16(lo) | uint16(hi)<<8
}

// TODO UNTESTED HELPER FUNCTION
func (gsu *GSU) ramWordStore(addr uint16, register byte, isByte, isWriteback bool) {
	var bank byte
	if isWriteback {
		bank, addr = byte(gsu.prevRamAddr>>16), uint16(gsu.prevRamAddr)
	} else {
		bank = SRAM_BASE_BANK + gsu.r.RAMBR
		gsu.prevRamAddr = uint32(bank)<<16 | uint32(addr)
	}

	gsu.Write8(bank, addr, byte(gsu.r.cpuRegisters[register]))
	if isByte {
		gsu.Write8(bank, addr^1, byte(gsu.r.cpuRegisters[register]>>8))
	}
}

func branchFunc(gsu *GSU) {
	var shouldBranch bool
	switch gsu.immediateOpcode {
	case 0x05:
		shouldBranch = true
	case 0x06:
		shouldBranch = min(1, gsu.r.SFR&FlagS)^min(1, gsu.r.SFR&FlagV) == 0
	case 0x07:
		shouldBranch = min(1, gsu.r.SFR&FlagS)^min(1, gsu.r.SFR&FlagV) == 1
	case 0x08:
		shouldBranch = gsu.r.SFR&FlagZ == 0
	case 0x09:
		shouldBranch = gsu.r.SFR&FlagZ == 1
	case 0x0A:
		shouldBranch = gsu.r.SFR&FlagS == 0
	case 0x0B:
		shouldBranch = gsu.r.SFR&FlagS == 1
	case 0x0C:
		shouldBranch = gsu.r.SFR&FlagC == 0
	case 0x0D:
		shouldBranch = gsu.r.SFR&FlagC == 1
	case 0x0E:
		shouldBranch = gsu.r.SFR&FlagV == 0
	case 0x0F:
		shouldBranch = gsu.r.SFR&FlagV == 1
	}

	if shouldBranch {
		gsu.branchOffset = uint16(int8(gsu.immediateBytes[0]))
	}
	//DONT clear prefixes
}
