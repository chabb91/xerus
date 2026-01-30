package cpu

// prevents the 24 bit memory from overflow
func mask24(v uint32) uint32 { return v & 0x00FF_FFFF }

// merges a 16 bit offset with a page bank returning a full 24 bit memory address
func mapOffsetToBank(bank byte, addr uint16) uint32 {
	return (uint32(bank) << 16) | uint32(addr)
}

// Relative 8-bit: target PC = (PC + int8(op1))
// rel8 is tied to is PB and can never cross it. however it can cross pages
func rel8(cpu *CPU, disp byte) {
	cpu.r.PC += uint16(int8(disp))
}

// Relative 16-bit (BRL): 16-bit signed displacement
// wraps at bank boundary so it jumps anywhere in the current PB
func rel16(val uint16) uint16 {
	return uint16(int16(val))
}

// Most likely the proper addressing logic for the Direct Page mode.
// One main functions and 2 wrappers for convenience
func directPageLogic(cpu *CPU, op byte, register uint16, isPEI bool) (addressLo, addressHi uint32) {
	if cpu.isW() && cpu.r.E && !isPEI {
		low := getLowByte(uint16(op) + register)
		addressLo = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low))
		addressHi = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low+1))
	} else {
		offset := cpu.r.D + uint16(op) + register
		addressLo = mapOffsetToBank(0x00, offset)
		addressHi = mapOffsetToBank(0x00, offset+1)
	}

	return addressLo, addressHi
}

func directPage(cpu *CPU, op byte, isPEI bool) (addressLo, addressHi uint32) {
	addressLo, addressHi = directPageLogic(cpu, op, 0, isPEI)
	return addressLo, addressHi
}

func directPageXY(cpu *CPU, op byte, register uint16, mode AddressingMode) (addressLo, addressHi uint32) {
	addressLo, addressHi = directPageLogic(cpu, op, register, false)
	//little hardware quirk
	//the last part is just the !isW function but not slow
	//TODO remove isW and optimize it instead
	if mode == INDEXED_INDIRECT && cpu.r.E && (cpu.r.D&0xFF != 0) {
		addressHi = (addressLo+1)&0xFF | addressLo&0xFFFF00
	}
	return addressLo, addressHi
}

func directPageLong(cpu *CPU, op byte) (addressLo, addressHi, addressBank uint32) {
	/*
		if cpu.isW() && cpu.r.E {
			low := op
			addressLo = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low))
			addressHi = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low+1))
			addressBank = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low+2))
		} else {
			offset := cpu.r.D + uint16(op)
			addressLo = mapOffsetToBank(0x00, offset)
			addressHi = mapOffsetToBank(0x00, offset+1)
			addressBank = mapOffsetToBank(0x00, offset+2)
		}
	*/
	// TODO: Test data shows inconsistent direct page wrapping behavior:
	// [dir],Y wraps when DL==0, [dir] doesn't wrap
	// Need to verify with full program tests before implementing
	offset := cpu.r.D + uint16(op)
	addressLo = mapOffsetToBank(0x00, offset)
	addressHi = mapOffsetToBank(0x00, offset+1)
	addressBank = mapOffsetToBank(0x00, offset+2)
	return addressLo, addressHi, addressBank
}

func absoluteXY(bank, high, low byte, register uint16) (addressLo, addressHi uint32) {
	addressLo = mask24(mapOffsetToBank(bank, createWord(high, low)) + uint32(register))
	addressHi = mask24(addressLo + 1)
	return addressLo, addressHi
}

// Absolute: [DBR: op1|op2]
func absolute(bank, high, low byte) (uint32, uint32) {
	return absoluteXY(bank, high, low, 0)
}
