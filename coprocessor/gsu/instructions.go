package gsu

import (
	"fmt"
)

func (gsu *GSU) processByte() {
	if immediateNum := gsu.r.SFR.getImmediateNum(); immediateNum != 0 {
		immediateNum--
		gsu.immediateBytes[immediateNum] = gsu.currentOpcode
		gsu.r.SFR.setImmediateNum(immediateNum)
		if immediateNum == 0 {
			gsu.immediateInstruction(gsu)
		}
	} else {
		switch opcode := gsu.currentOpcode; opcode {
		case 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F: //BRANCH
			gsu.r.SFR.setImmediateNum(1)
			gsu.immediateInstruction = branchFunc
			gsu.immediateOpcode = opcode
		case 0xF0, 0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, //IWT
			0xFB, 0xFC, 0xFD, 0xFE, 0xFF:
			gsu.r.SFR.setImmediateNum(2)
			gsu.immediateInstruction = iwtFunc
			gsu.immediateOpcode = opcode
		case 0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, //IBT
			0xAB, 0xAC, 0xAD, 0xAE, 0xAF:
			gsu.r.SFR.setImmediateNum(1)
			gsu.immediateInstruction = ibtFunc
			gsu.immediateOpcode = opcode
		case 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, //STW
			0x3B:
			gsu.ramWordStore(gsu.r.cpuRegisters[opcode&0x0F], gsu.sReg, gsu.r.SFR.getAltNum() == FlagAlt1, false)
			gsu.clearPrefixes()
		case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, //LDW
			0x4B:
			gsu.ramWordLoad(gsu.r.cpuRegisters[opcode&0x0F], gsu.dReg, gsu.r.SFR.getAltNum() == FlagAlt1)
			gsu.clearPrefixes()
		case 0x90: //SBK
			gsu.ramWordStore(0, gsu.sReg, false, true)
			gsu.clearPrefixes()
		case 0xEF: //GET(load byte from rom)
			byte := gsu.readRomAddrPtr()
			switch gsu.r.SFR.getAltNum() {
			case FlagAlt1: // GETBH
				rs := gsu.r.cpuRegisters[gsu.sReg]
				gsu.r.writeCpuRegister(gsu.dReg, (uint16(byte)<<8)|(rs&0x00FF))
			case FlagAlt2: // GETBL
				rs := gsu.r.cpuRegisters[gsu.sReg]
				gsu.r.writeCpuRegister(gsu.dReg, (rs&0xFF00)|uint16(byte))
			case FlagAlt3: // GETBS
				//technically the inner int16 cast isnt needed, but cant test
				gsu.r.writeCpuRegister(gsu.dReg, uint16(int16(int8(byte))))
			default: // GETB
				gsu.r.writeCpuRegister(gsu.dReg, uint16(byte))
			}
			gsu.clearPrefixes()
		case 0xDF: //GETC pretending as RAMB/ROMB
			switch gsu.r.SFR.getAltNum() {
			case FlagAlt2:
				gsu.r.RAMBR = byte(gsu.r.cpuRegisters[gsu.sReg]) & 1
			case FlagAlt3:
				gsu.r.ROMBR = byte(gsu.r.cpuRegisters[gsu.sReg])
			default:
				color := gsu.readRomAddrPtr()
				gsu.r.setColr(color)
			}
			gsu.clearPrefixes()
		case 0x4E: //COLOR/CMODE
			if gsu.r.SFR.getAltNum() == FlagAlt1 {
				gsu.r.POR = por(gsu.r.cpuRegisters[gsu.sReg]) & 0x1F
			} else {
				gsu.r.setColr(byte(gsu.r.cpuRegisters[gsu.sReg]))
			}
			gsu.clearPrefixes()
		case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, //ADD/ADC
			0x5B, 0x5C, 0x5D, 0x5E, 0x5F:
			augend := gsu.r.cpuRegisters[gsu.sReg]
			result32 := uint32(augend)

			if gsu.r.SFR&FlagAlt1 != 0 { //adc
				result32 += uint32(min(gsu.r.SFR&FlagC, 1))
			}
			addend := uint16(opcode & 0x0F)
			if gsu.r.SFR&FlagAlt2 == 0 {
				addend = gsu.r.cpuRegisters[addend]
			}
			result32 += uint32(addend)

			result := uint16(result32)
			setFlag(&gsu.r.SFR, FlagC, result32 > 0xFFFF)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			setFlag(&gsu.r.SFR, FlagV, (result^augend)&(result^addend)&0x8000 != 0)
			//setFlag(&gsu.r.SFR, FlagV, result&0x8000 != signA && signA == signB)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6A, //SUB/SBC//CMP
			0x6B, 0x6C, 0x6D, 0x6E, 0x6F:
			minuend := uint16(gsu.r.cpuRegisters[gsu.sReg])
			result32 := uint32(minuend)

			alt := gsu.r.SFR.getAltNum()

			subtrahend := uint16(opcode & 0x0F)
			if alt != FlagAlt2 {
				subtrahend = gsu.r.cpuRegisters[subtrahend]
			}
			result32 -= uint32(subtrahend)

			if alt == FlagAlt1 {
				result32 -= (uint32(min(1, gsu.r.SFR&FlagC) ^ 1))
			}

			result := uint16(result32)
			setFlag(&gsu.r.SFR, FlagC, result32 <= 0xFFFF)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			setFlag(&gsu.r.SFR, FlagV, (minuend^subtrahend)&(minuend^result)&0x8000 != 0)
			//subtraction overflow: if i subtract 30000 -(-30000) its expected to be large positive
			//if its negative -> overflow
			//setFlag(&gsu.r.SFR, FlagV, result&0x8000 != signA && signA != signB)
			if alt != FlagAlt3 {
				gsu.r.writeCpuRegister(gsu.dReg, result)
			}
			gsu.clearPrefixes()
		case 0x70: //MERGE
			result := gsu.r.cpuRegisters[7]&0xFF00 | gsu.r.cpuRegisters[8]>>8
			setFlag(&gsu.r.SFR, FlagC, result&0xE0E0 != 0)
			setFlag(&gsu.r.SFR, FlagZ, result&0xF0F0 != 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8080 != 0)
			setFlag(&gsu.r.SFR, FlagV, result&0xC0C0 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7A, 0x7B, //AND/BIC
			0x7C, 0x7D, 0x7E, 0x7F:
			operand := uint16(opcode & 0x0F)
			if gsu.r.SFR&FlagAlt2 == 0 {
				operand = gsu.r.cpuRegisters[operand]
			}
			if gsu.r.SFR&FlagAlt1 != 0 {
				operand = ^operand
			}
			result := gsu.r.cpuRegisters[gsu.sReg] & operand
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0xC0: //HIB
			result := gsu.r.cpuRegisters[gsu.sReg] >> 8
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x80 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, //OR/XOR
			0xCC, 0xCD, 0xCE, 0xCF:
			operand := uint16(opcode & 0x0F)
			if gsu.r.SFR&FlagAlt2 == 0 {
				operand = gsu.r.cpuRegisters[operand]
			}
			result := gsu.r.cpuRegisters[gsu.sReg]
			if gsu.r.SFR&FlagAlt1 != 0 {
				result ^= operand
			} else {
				result |= operand
			}
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x4F: //NOT
			result := ^gsu.r.cpuRegisters[gsu.sReg]
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7, 0xD8, 0xD9, 0xDA, //INC
			0xDB, 0xDC, 0xDD, 0xDE:
			reg := opcode & 0x0F
			result := gsu.r.cpuRegisters[reg] + 1
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(reg, result)
			gsu.clearPrefixes()
		case 0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, //DEC
			0xEB, 0xEC, 0xED, 0xEE:
			reg := opcode & 0x0F
			result := gsu.r.cpuRegisters[reg] - 1
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(reg, result)
			gsu.clearPrefixes()
		case 0x03: //LSR
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			result >>= 1
			setFlag(&gsu.r.SFR, FlagC, lsb != 0)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, false)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x04: //ROL
			result := gsu.r.cpuRegisters[gsu.sReg]
			msb := result & 0x8000
			result = gsu.r.cpuRegisters[gsu.sReg]<<1 | uint16(min(gsu.r.SFR&FlagC, 1))
			setFlag(&gsu.r.SFR, FlagC, msb != 0)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x96: //ASR -signed shift
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			if gsu.r.SFR.getAltNum() == FlagAlt1 && result == 0xFFFF {
				result = 0 //DIV 2: -1>>1 == -1. needs a little push
			}
			result = uint16(int16(result) >> 1)
			setFlag(&gsu.r.SFR, FlagC, lsb != 0)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x97: //ROR
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			result = gsu.r.cpuRegisters[gsu.sReg]>>1 | uint16(min(gsu.r.SFR&FlagC, 1)<<15)
			setFlag(&gsu.r.SFR, FlagC, lsb != 0)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x4D: //SWAP
			result := gsu.r.cpuRegisters[gsu.sReg]
			result = result<<8 | result>>8
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x95: //SEX
			result := uint16(int8(gsu.r.cpuRegisters[gsu.sReg] & 0xFF))
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x9E: //LOB
			result := gsu.r.cpuRegisters[gsu.sReg]
			result &= 0xFF
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x80 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case 0x9F: //FMULT/LMULT
			altNum := gsu.r.SFR.getAltNum()
			result32 := uint32(int32(int16(gsu.r.cpuRegisters[gsu.sReg])) * int32(int16(gsu.r.cpuRegisters[6])))
			if altNum == FlagAlt1 {
				//if dreg == 4 this is obviously overwritten
				gsu.r.writeCpuRegister(4, uint16(result32))
			}
			result := uint16(result32 >> 16)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			setFlag(&gsu.r.SFR, FlagC, result32&0x8000 != 0)
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result32&0x8000_0000 != 0)
			gsu.stepMultiplication(true)
			gsu.clearPrefixes()
		case 0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8A, //MULT/UMULT
			0x8B, 0x8C, 0x8D, 0x8E, 0x8F:
			result := uint16(gsu.r.cpuRegisters[gsu.sReg] & 0x00FF)
			multiplier := uint16(opcode & 0x0F)
			if gsu.r.SFR&FlagAlt2 == 0 {
				multiplier = gsu.r.cpuRegisters[multiplier] & 0x00FF
			}
			if gsu.r.SFR&FlagAlt1 == 0 {
				result = uint16(int16(int8(result)) * int16(int8(multiplier)))
			} else {
				result *= multiplier
			}
			setFlag(&gsu.r.SFR, FlagZ, result == 0)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.stepMultiplication(false)
			gsu.clearPrefixes() //JMP/LJMP
		case 0x98, 0x99, 0x9A, 0x9B, 0x9C, 0x9D:
			if gsu.r.SFR.getAltNum() == FlagAlt1 {
				gsu.r.PBR = byte(gsu.r.cpuRegisters[opcode&0x0F]) & 0x7F
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[gsu.sReg])
				gsu.r.CBR = gsu.r.cpuRegisters[gsu.sReg] & 0xFFF0
				gsu.cacheFlags = 0
			} else {
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[opcode&0x0F])
			}
			gsu.clearPrefixes()
		case 0x3C: //LOOP
			result := gsu.r.cpuRegisters[12] - 1
			isZero := result == 0
			setFlag(&gsu.r.SFR, FlagZ, isZero)
			setFlag(&gsu.r.SFR, FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(12, result)
			if !isZero {
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[13])
			}
			gsu.clearPrefixes() //LINK/RETURN TO
		case 0x91, 0x92, 0x93, 0x94:
			gsu.r.writeCpuRegister(11, gsu.r.cpuRegisters[0xF]+uint16(opcode&0x0F))
			gsu.clearPrefixes()
		case 0x3D: //ALT1
			gsu.r.SFR |= FlagAlt1
			gsu.r.SFR &= ^FlagB
		case 0x3E: //ALT2
			gsu.r.SFR |= FlagAlt2
			gsu.r.SFR &= ^FlagB
		case 0x3F: //ALT3
			gsu.r.SFR |= (FlagAlt1 | FlagAlt2)
			gsu.r.SFR &= ^FlagB
		case 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, //TO
			0x1B, 0x1C, 0x1D, 0x1E, 0x1F:
			dReg := opcode & 0x0F
			if gsu.r.SFR&FlagB != 0 { //MOVE
				gsu.r.writeCpuRegister(dReg, gsu.r.cpuRegisters[gsu.sReg])
				gsu.clearPrefixes()
			} else {
				gsu.dReg = dReg
			}
		case 0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, //FROM
			0xBB, 0xBC, 0xBD, 0xBE, 0xBF:
			sReg := opcode & 0x0F
			if gsu.r.SFR&FlagB != 0 { //MOVES
				val := gsu.r.cpuRegisters[sReg]
				gsu.r.writeCpuRegister(gsu.dReg, val)
				setFlag(&gsu.r.SFR, FlagZ, val == 0)
				setFlag(&gsu.r.SFR, FlagS, val&0x8000 != 0)
				setFlag(&gsu.r.SFR, FlagV, val&0x80 != 0)
				gsu.clearPrefixes()
			} else {
				gsu.sReg = sReg
			}
		case 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, //WITH
			0x2B, 0x2C, 0x2D, 0x2E, 0x2F:
			gsu.r.SFR |= FlagB

			reg := opcode & 0x0F
			gsu.sReg, gsu.dReg = reg, reg
		case 0x00: //STOP
			gsu.r.SFR &= ^FlagGo
			gsu.r.SFR |= FlagIrq
			if !hasFlag(gsu.r.CFGR, MaskIrq) {
				gsu.interruptManager.CartFireIrq()
			}
			gsu.clearPrefixes()
		case 0x01: //NOP
			gsu.clearPrefixes()
		case 0x02: //CACHE
			if cbr := gsu.r.cpuRegisters[0xF] & 0xFFF0; gsu.r.CBR != cbr {
				gsu.r.CBR = cbr
				//flush cache??
				gsu.cacheFlags = 0
			}
			gsu.clearPrefixes()
		case 0x4C: //PLOT
			x := gsu.r.cpuRegisters[1]
			y := gsu.r.cpuRegisters[2]
			if gsu.r.SFR.getAltNum() == FlagAlt1 { //RPIX
				data := gsu.rpix(x, y)
				setFlag(&gsu.r.SFR, FlagZ, data == 0)
				setFlag(&gsu.r.SFR, FlagS, uint16(data)&0x8000 != 0)
				gsu.r.writeCpuRegister(gsu.dReg, uint16(data))
			} else {
				gsu.plot(byte(x), byte(y))
				gsu.r.cpuRegisters[1]++
			}
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
	switch gsu.r.SFR.getAltNum() {
	case FlagAlt1:
		gsu.ramWordLoad(hilo, reg, false)
	case FlagAlt2:
		gsu.ramWordStore(hilo, reg, false, false)
	default:
		gsu.r.writeCpuRegister(reg, hilo)
	}
	gsu.clearPrefixes()
}

func ibtFunc(gsu *GSU) {
	reg := gsu.immediateOpcode & 0xF
	kk := gsu.immediateBytes[0]
	switch gsu.r.SFR.getAltNum() {
	case FlagAlt1:
		gsu.ramWordLoad(uint16(kk)<<1, reg, false)
	case FlagAlt2:
		gsu.ramWordStore(uint16(kk)<<1, reg, false, false)
	default:
		gsu.r.writeCpuRegister(reg, uint16(int8(kk)))
	}
	gsu.clearPrefixes()
}

func (gsu *GSU) ramWordLoad(addr uint16, register byte, isByte bool) {
	bank := SRAM_BASE_BANK + gsu.r.RAMBR
	gsu.prevRamAddr = uint32(bank)<<16 | uint32(addr)

	lo, _ := gsu.Read8(bank, addr)
	gsu.stepCart()
	hi := byte(0)
	if !isByte {
		hi, _ = gsu.Read8(bank, addr^1)
		gsu.stepCart()
	}
	gsu.r.writeCpuRegister(register, uint16(lo)|uint16(hi)<<8)
}

func (gsu *GSU) ramWordStore(addr uint16, register byte, isByte, isWriteback bool) {
	var bank byte
	if isWriteback {
		bank, addr = byte(gsu.prevRamAddr>>16), uint16(gsu.prevRamAddr)
	} else {
		bank = SRAM_BASE_BANK + gsu.r.RAMBR
		gsu.prevRamAddr = uint32(bank)<<16 | uint32(addr)
	}
	gsu.waitRamWriteCacheFlush()

	gsu.Write8(bank, addr, byte(gsu.r.cpuRegisters[register]))
	gsu.incrementRamWriteCacheClock()
	if !isByte {
		gsu.Write8(bank, addr^1, byte(gsu.r.cpuRegisters[register]>>8))
		gsu.incrementRamWriteCacheClock()
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
		shouldBranch = !hasFlag(gsu.r.SFR, FlagZ)
	case 0x09:
		shouldBranch = hasFlag(gsu.r.SFR, FlagZ)
	case 0x0A:
		shouldBranch = !hasFlag(gsu.r.SFR, FlagS)
	case 0x0B:
		shouldBranch = hasFlag(gsu.r.SFR, FlagS)
	case 0x0C:
		shouldBranch = !hasFlag(gsu.r.SFR, FlagC)
	case 0x0D:
		shouldBranch = hasFlag(gsu.r.SFR, FlagC)
	case 0x0E:
		shouldBranch = !hasFlag(gsu.r.SFR, FlagV)
	case 0x0F:
		shouldBranch = hasFlag(gsu.r.SFR, FlagV)
	}

	if shouldBranch {
		gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[0xF]+uint16(int8(gsu.immediateBytes[0])))
	}
	//DONT clear prefixes
}
