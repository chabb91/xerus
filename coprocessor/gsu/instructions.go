package gsu

import "fmt"

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
		switch opcode := gsu.fetchedByte; {
		case opcode-5 <= 0xA: //BRANCH instructions 0x05-0x0F UNTESTED
			gsu.r.setImmediateNum(1)
			gsu.immediateInstruction = branchFunc
			gsu.immediateOpcode = opcode
		case opcode&0xF0 == 0xF0: //IWT instructions
			gsu.r.setImmediateNum(2)
			gsu.immediateInstruction = iwtFunc
			gsu.immediateOpcode = opcode
		case opcode&0xF0 == 0xA0: //IBT instructions
			gsu.r.setImmediateNum(1)
			gsu.immediateInstruction = ibtFunc
			gsu.immediateOpcode = opcode
		case opcode&0xF0 == 0x50: //ADD/ADC instructions
			reg := opcode & 0xF
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
		case opcode&0xF0 == 0x10: //TO
			gsu.dReg = opcode & 0xF
		case opcode&0xF0 == 0xB0: //FROM
			gsu.sReg = opcode & 0xF
		case opcode&0xF0 == 0x20: //WITH
			gsu.r.SFR |= FlagB

			reg := opcode & 0xF
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
	//TODO:ALT1 and ALT1 UNTESTED. these modes also run 9/11 cycles so gotta model that too
	case 1:
		hilo &= 0xFFFE
		lo, _ := gsu.Read8(0x70+gsu.r.RAMBR, hilo)
		hi, _ := gsu.Read8(0x70+gsu.r.RAMBR, hilo+1)
		gsu.r.cpuRegisters[reg] = uint16(lo) | uint16(hi)<<8
	case 2:
		hilo &= 0xFFFE
		gsu.Write8(0x70+gsu.r.RAMBR, hilo, byte(gsu.r.cpuRegisters[reg]))
		gsu.Write8(0x70+gsu.r.RAMBR, hilo+1, byte(gsu.r.cpuRegisters[reg]>>8))
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
	//TODO:ALT1 and ALT1 UNTESTED. these modes also run 10/8 cycles so gotta model that too
	case 1:
		hilo := uint16(kk) << 1
		lo, _ := gsu.Read8(0x70+gsu.r.RAMBR, hilo)
		hi, _ := gsu.Read8(0x70+gsu.r.RAMBR, hilo+1)
		gsu.r.cpuRegisters[reg] = uint16(lo) | uint16(hi)<<8
	case 2:
		hilo := uint16(kk) << 1
		gsu.Write8(0x70+gsu.r.RAMBR, hilo, byte(gsu.r.cpuRegisters[reg]))
		gsu.Write8(0x70+gsu.r.RAMBR, hilo+1, byte(gsu.r.cpuRegisters[reg]>>8))
	default:
		gsu.r.cpuRegisters[reg] = uint16(int8(kk))
		fmt.Println("IBT normal mode")
	}
	gsu.clearPrefixes()
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
