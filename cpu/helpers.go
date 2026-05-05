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
		low := op + byte(register)
		addressLo = mapOffsetToBank(0x00, cpu.r.D|uint16(low))
		addressHi = mapOffsetToBank(0x00, cpu.r.D|uint16(low+1))
	} else {
		offset := cpu.r.D + uint16(op) + register
		addressLo = mapOffsetToBank(0x00, offset)
		addressHi = mapOffsetToBank(0x00, offset+1)
	}
	return
}

func absolute(bank, high, low byte, register uint16) (addressLo, addressHi uint32) {
	addressLo = mask24(mapOffsetToBank(bank, createWord(high, low)) + uint32(register))
	addressHi = mask24(addressLo + 1)
	return
}

// high byte=AB of ABCD
func getHighByte(fullValue uint16) byte {
	return byte((0xFF00 & fullValue) >> 8)
}

// low byte=CD of ABCD
func getLowByte(fullValue uint16) byte {
	return byte(0x00FF & fullValue)
}

func isPageBoundaryCrossed(addr1, addr2 uint16) bool {
	return (addr1 & 0xFF00) != (addr2 & 0xFF00)
}

// Merges two bytes into a single uint16 or word.
// hi:lo is returned
func createWord(high, low byte) uint16 {
	return (uint16(high) << 8) | uint16(low)
}

// Merges three bytes into a single uint32 representing a full 24 bit address.
// bank:hi:lo is returned
func createAddress(low, high, bank byte) uint32 {
	return uint32(low) | uint32(high)<<8 | uint32(bank)<<16
}

// splits a word into two bytes
// hi,lo is returned
func splitWord(word uint16) (high, low byte) {
	high = byte(word >> 8)
	low = byte(word & 0xFF)

	return
}

// check if the low byte of the D register is 0
// needed for cycle calculations and for a legacy edge case in direct page addressing
func (cpu *CPU) isW() bool {
	return cpu.r.D&0xFF == 0
}

// if M or X flags are 1 and indicating 8 bit mode return the 8 as in how many bits are we working on
func getInstructionWidth(c bool) iWidth {
	if c {
		return W_8
	} else {
		return W_16
	}
}
