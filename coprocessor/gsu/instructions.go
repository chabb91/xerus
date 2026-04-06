package gsu

import (
	"fmt"
)

func (gsu *GSU) processByte() {
	immediateNum := gsu.r.getImmediateNum()
	if immediateNum != 0 {
		immediateNum--
		gsu.immediateBytes[immediateNum] = gsu.currentOpcode
		gsu.r.setImmediateNum(immediateNum)
		if immediateNum == 0 {
			gsu.immediateInstruction(gsu)
		}
		return
	} else {
		opcode := gsu.currentOpcode
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
			switch gsu.r.getAltNum() {
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
		case opcode == 0xDF: //GETC pretending as RAMB/ROMB
			switch gsu.r.getAltNum() {
			case FlagAlt2:
				gsu.r.RAMBR = byte(gsu.r.cpuRegisters[gsu.sReg]) & 1
			case FlagAlt3:
				gsu.r.ROMBR = byte(gsu.r.cpuRegisters[gsu.sReg])
			default:
				color, _ := gsu.Read8(gsu.r.ROMBR, gsu.r.cpuRegisters[14])
				gsu.r.setColr(color)
			}
			gsu.clearPrefixes()
		case opcode == 0x4E: //COLOR/CMODE
			if gsu.r.getAltNum() == FlagAlt1 {
				gsu.r.POR = byte(gsu.r.cpuRegisters[gsu.sReg]) & 0x1F
			} else {
				gsu.r.setColr(byte(gsu.r.cpuRegisters[gsu.sReg]))
			}
			gsu.clearPrefixes()
		case opcodeHn == 0x50: //ADD/ADC instructions
			augend := gsu.r.cpuRegisters[gsu.sReg]
			result32 := uint32(augend)

			if gsu.r.SFR&FlagAlt1 != 0 { //adc
				result32 += uint32(min(gsu.r.SFR&FlagC, 1))
			}
			addend := uint16(opcodeLn)
			if gsu.r.SFR&FlagAlt2 == 0 {
				addend = gsu.r.cpuRegisters[opcodeLn]
			}
			result32 += uint32(addend)

			result := uint16(result32)
			gsu.r.setFlag(FlagC, result32 > 0xFFFF)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.setFlag(FlagV, (result^augend)&(result^addend)&0x8000 != 0)
			//gsu.r.setFlag(FlagV, result&0x8000 != signA && signA == signB)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcodeHn == 0x60: //SUB/SBC//CMP instructions
			minuend := uint16(gsu.r.cpuRegisters[gsu.sReg])
			result32 := uint32(minuend)

			alt := gsu.r.getAltNum()

			subtrahend := uint16(opcodeLn)
			if alt != FlagAlt2 {
				subtrahend = gsu.r.cpuRegisters[opcodeLn]
			}
			result32 -= uint32(subtrahend)

			if alt == FlagAlt1 {
				result32 -= (uint32(min(1, gsu.r.SFR&FlagC) ^ 1))
			}

			result := uint16(result32)
			gsu.r.setFlag(FlagC, result32 <= 0xFFFF)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.setFlag(FlagV, (minuend^subtrahend)&(minuend^result)&0x8000 != 0)
			//subtraction overflow = if i subtract 30000 -(-30000) its expected to be large positive
			//if its negative -> overflow
			//gsu.r.setFlag(FlagV, result&0x8000 != signA && signA != signB)
			if alt != FlagAlt3 {
				gsu.r.writeCpuRegister(gsu.dReg, result)
			}
			gsu.clearPrefixes()
		case opcode == 0x70: //MERGE
			result := gsu.r.cpuRegisters[7]&0xFF00 | gsu.r.cpuRegisters[8]>>8
			gsu.r.setFlag(FlagC, result&0xE0E0 != 0)
			gsu.r.setFlag(FlagZ, result&0xF0F0 != 0)
			gsu.r.setFlag(FlagS, result&0x8080 != 0)
			gsu.r.setFlag(FlagV, result&0xC0C0 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode-0x71 <= 0xE: //AND/BIC
			operand := uint16(opcodeLn)
			if gsu.r.SFR&FlagAlt2 == 0 {
				operand = gsu.r.cpuRegisters[opcodeLn]
			}
			if gsu.r.SFR&FlagAlt1 != 0 {
				operand = ^operand
			}
			result := gsu.r.cpuRegisters[gsu.sReg] & operand
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0xC0: //HIB
			result := gsu.r.cpuRegisters[gsu.sReg] >> 8
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x80 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode-0xC1 <= 0xE: //OR/XOR
			operand := uint16(opcodeLn)
			if gsu.r.SFR&FlagAlt2 == 0 {
				operand = gsu.r.cpuRegisters[opcodeLn]
			}
			result := gsu.r.cpuRegisters[gsu.sReg]
			if gsu.r.SFR&FlagAlt1 != 0 {
				result ^= operand
			} else {
				result |= operand
			}
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x4F: //NOT
			result := ^gsu.r.cpuRegisters[gsu.sReg]
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode-0xD0 <= 0xE: //INC
			result := gsu.r.cpuRegisters[opcodeLn] + 1
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(opcodeLn, result)
			gsu.clearPrefixes()
		case opcode-0xE0 <= 0xE: //DEC
			result := gsu.r.cpuRegisters[opcodeLn] - 1
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(opcodeLn, result)
			gsu.clearPrefixes()
		case opcode == 0x03: //LSR
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			result >>= 1
			gsu.r.setFlag(FlagC, lsb != 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, false)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x04: //ROL
			result := gsu.r.cpuRegisters[gsu.sReg]
			msb := result & 0x8000
			result = gsu.r.cpuRegisters[gsu.sReg]<<1 | min(gsu.r.SFR&FlagC, 1)
			gsu.r.setFlag(FlagC, msb != 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x96: //ASR -signed shift
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			if gsu.r.getAltNum() == FlagAlt1 && result == 0xFFFF {
				result = 0 //DIV 2: -1>>1 == -1. needs a little push
			}
			result = uint16(int16(result) >> 1)
			gsu.r.setFlag(FlagC, lsb != 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x97: //ROR
			result := gsu.r.cpuRegisters[gsu.sReg]
			lsb := result & 1
			result = gsu.r.cpuRegisters[gsu.sReg]>>1 | min(gsu.r.SFR&FlagC, 1)<<15
			gsu.r.setFlag(FlagC, lsb != 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x4D: //SWAP
			result := gsu.r.cpuRegisters[gsu.sReg]
			result = result<<8 | result>>8
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x95: //SEX
			result := uint16(int8(gsu.r.cpuRegisters[gsu.sReg] & 0xFF))
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x9E: //LOB
			result := gsu.r.cpuRegisters[gsu.sReg]
			result &= 0xFF
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x80 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode == 0x9F: //FMULT/LMULT
			altNum := gsu.r.getAltNum()
			result32 := uint32(int32(int16(gsu.r.cpuRegisters[gsu.sReg])) * int32(int16(gsu.r.cpuRegisters[6])))
			if altNum == FlagAlt1 {
				//if dreg == 4 this is obviously overwritten
				gsu.r.writeCpuRegister(4, uint16(result32))
			}
			result := uint16(result32 >> 16)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.r.setFlag(FlagC, result32&0x8000 != 0)
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result32&0x8000_0000 != 0)
			gsu.clearPrefixes()
		case opcodeHn == 0x80: //MULT/UMULT
			result := uint16(gsu.r.cpuRegisters[gsu.sReg] & 0x00FF)
			multiplier := uint16(opcodeLn)
			if gsu.r.SFR&FlagAlt2 == 0 {
				multiplier = gsu.r.cpuRegisters[opcodeLn] & 0x00FF
			}
			if gsu.r.SFR&FlagAlt1 == 0 {
				result = uint16(int16(int8(result)) * int16(int8(multiplier)))
			} else {
				result *= multiplier
			}
			gsu.r.setFlag(FlagZ, result == 0)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(gsu.dReg, result)
			gsu.clearPrefixes()
		case opcode-0x98 <= 5: //JMP/LJMP
			if gsu.r.getAltNum() == FlagAlt1 {
				gsu.r.PBR = byte(gsu.r.cpuRegisters[opcodeLn]) & 0x7F
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[gsu.sReg])
				gsu.r.CBR = gsu.r.cpuRegisters[gsu.sReg] & 0xFFF0
				//TODO flush cache
				gsu.cacheFlags = 0
			} else {
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[opcodeLn])
			}
			gsu.clearPrefixes()
		case opcode == 0x3C: //LOOP
			result := gsu.r.cpuRegisters[12] - 1
			isZero := result == 0
			gsu.r.setFlag(FlagZ, isZero)
			gsu.r.setFlag(FlagS, result&0x8000 != 0)
			gsu.r.writeCpuRegister(12, result)
			if !isZero {
				gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[13])
			}
			gsu.clearPrefixes()
		case opcode-0x91 <= 3: //LINK/RETURN TO
			gsu.r.writeCpuRegister(11, gsu.r.cpuRegisters[0xF]+uint16(opcodeLn))
			gsu.clearPrefixes()
		case opcode == 0x3D: //ALT1
			gsu.r.SFR |= FlagAlt1
			gsu.r.SFR &= ^FlagB
		case opcode == 0x3E: //ALT2
			gsu.r.SFR |= FlagAlt2
			gsu.r.SFR &= ^FlagB
		case opcode == 0x3F: //ALT3
			gsu.r.SFR |= (FlagAlt1 | FlagAlt2)
			gsu.r.SFR &= ^FlagB
		case opcodeHn == 0x10: //TO
			dReg := opcodeLn
			if gsu.r.SFR&FlagB != 0 { //MOVE
				gsu.r.writeCpuRegister(dReg, gsu.r.cpuRegisters[gsu.sReg])
				gsu.clearPrefixes()
			} else {
				gsu.dReg = dReg
			}
		case opcodeHn == 0xB0: //FROM
			sReg := opcodeLn
			if gsu.r.SFR&FlagB != 0 { //MOVES
				val := gsu.r.cpuRegisters[sReg]
				gsu.r.writeCpuRegister(gsu.dReg, val)
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
		case opcode == 0x00: //STOP
			gsu.r.SFR &= ^FlagGo
			gsu.r.SFR |= FlagIrq
			gsu.clearPrefixes()
		case opcode == 0x01: //NOP
			gsu.clearPrefixes()
		case opcode == 0x02: //CACHE
			if cbr := gsu.r.cpuRegisters[0xF] & 0xFFF0; gsu.r.CBR != cbr {
				gsu.r.CBR = cbr
				//flush cache??
				gsu.cacheFlags = 0
			}
			gsu.clearPrefixes()
		case opcode == 0x4C: //PLOT??
			x := gsu.r.cpuRegisters[1]
			y := gsu.r.cpuRegisters[2]
			if gsu.r.gsu.r.getAltNum() == FlagAlt1 {
				//panic("GSU: RPIX NOT EVEN ONCE")
				tn := ((x & 0xF8) << 1) + ((y & 0xF8) >> 3)
				bpp := uint16(2)
				tra := uint32(tn)*(uint32(bpp)<<3) + gsu.r.SCBR + uint32((y&7)*2)
				x = (x & 7) ^ 7
				var data byte
				for i := range bpp {
					b := ((i >> 1) << 4) + (i & 1)
					addr2 := tra + uint32(b)
					val, _ := gsu.Read8(byte(addr2>>16), uint16(addr2))
					data |= ((val >> x) & 1) << i
				}
				gsu.r.setFlag(FlagZ, data == 0)
				gsu.r.setFlag(FlagS, uint16(data)&0x8000 != 0)
				gsu.r.writeCpuRegister(gsu.dReg, uint16(data))
			} else {
				tn := (x>>3)*0x10 + (y >> 3)
				tra := uint32(tn)*0x10 + uint32(gsu.r.SCBR) + uint32(y&7)<<1
				col := (x & 7) ^ 7
				bp0, _ := gsu.Read8(byte(tra>>16), uint16(tra))
				bp1, _ := gsu.Read8(byte(tra>>16), uint16(tra)+1)
				bp0 &= ^(1 << col)
				bp0 |= gsu.r.COLR & 1 << col
				bp1 &= ^(1 << col)
				bp1 |= (gsu.r.COLR >> 1) & 1 << col
				gsu.Write8(byte(tra>>16), uint16(tra), bp0)
				gsu.Write8(byte(tra>>16), uint16(tra)+1, bp1)
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
	switch gsu.r.getAltNum() {
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
	switch gsu.r.getAltNum() {
	case FlagAlt1:
		gsu.ramWordLoad(uint16(kk)<<1, reg, false)
	case FlagAlt2:
		gsu.ramWordStore(uint16(kk)<<1, reg, false, false)
	default:
		gsu.r.writeCpuRegister(reg, uint16(int8(kk)))
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
	gsu.r.writeCpuRegister(register, uint16(lo)|uint16(hi)<<8)
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
	if !isByte {
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
		shouldBranch = gsu.r.SFR&FlagZ != 0
	case 0x0A:
		shouldBranch = gsu.r.SFR&FlagS == 0
	case 0x0B:
		shouldBranch = gsu.r.SFR&FlagS != 0
	case 0x0C:
		shouldBranch = gsu.r.SFR&FlagC == 0
	case 0x0D:
		shouldBranch = gsu.r.SFR&FlagC != 0
	case 0x0E:
		shouldBranch = gsu.r.SFR&FlagV == 0
	case 0x0F:
		shouldBranch = gsu.r.SFR&FlagV != 0
	}

	if shouldBranch {
		gsu.r.writeCpuRegister(0xF, gsu.r.cpuRegisters[0xF]+uint16(int8(gsu.immediateBytes[0])))
	}
	//DONT clear prefixes
}
