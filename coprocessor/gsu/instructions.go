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
		case opcode&0xF0 == 0xF0: //IWT instructions
			gsu.r.setImmediateNum(2)
			gsu.immediateInstruction = iwtFunc
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
		}
	}
}

func (gsu *GSU) clearPrefixes() {
	gsu.r.SFR &= ^(FlagB | FlagAlt1 | FlagAlt2)
	gsu.sReg, gsu.dReg = 0, 0
}

func iwtFunc(gsu *GSU) {
	reg := gsu.immediateOpcode & 0xF
	gsu.r.cpuRegisters[reg] = uint16(gsu.immediateBytes[0])<<8 | uint16(gsu.immediateBytes[1])
	gsu.clearPrefixes()
	fmt.Println("REG: ", reg, " :", gsu.r.cpuRegisters[reg])
}
