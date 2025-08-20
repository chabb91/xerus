package cpu

// SetLowByte takes a 16-bit value and an 8-bit value,
// and returns a new 16-bit value with the low byte updated.
// The high byte of the original value is preserved.
func SetLowByte(original *uint16, newLowByte byte) {
	*original = *original&0xFF00 | uint16(newLowByte)
}

// SetHighByte takes a 16-bit value and an 8-bit value,
// and returns a new 16-bit value with the high byte updated.
// The low byte of the original value is preserved.
func SetHighByte(original *uint16, newHighByte byte) {
	*original = *original&0x00FF | (uint16(newHighByte) << 8)
}

// high byte=AB of ABCD
func getHighByte(fullValue uint16) byte {
	return byte((0xFF00 & fullValue) >> 8)
}

// low byte=CD of ABCD
func getLowByte(fullValue uint16) byte {
	return byte(0x00FF & fullValue)
}

// masks CD of ABCD returning AB00
func maskLowByte(fullValue uint16) uint16 {
	return 0xFF00 & fullValue
}

// masks AB of ABCD returning 00CD
func maskHighByte(fullValue uint16) uint16 {
	return 0x00FF & fullValue
}

func isPageBoundaryCrossed(addr1, addr2 uint16) bool {
	return (addr1 & 0xFF00) != (addr2 & 0xFF00)
}

func isPageBoundaryCrossed24(addr1, addr2 uint32) bool {
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

	return high, low
}

func addByteToWordWithWrap(word uint16, b byte) uint16 {
	b += getLowByte(word)
	SetLowByte(&word, b)
	return word
}

// check if the low byte of the D register is 0
// needed for cycle calculations and for a legacy edge case in direct page addressing
func (cpu *CPU) isW() bool {
	return getLowByte(cpu.r.D) == 0
}

// if M or X flags are 1 and indicating 8 bit mode return the 8 as in how many bits are we working on
func boolToBitCount(c bool) int {
	if c {
		return 8
	} else {
		return 16
	}
}

func boolToFlag(b bool) byte {
	if b {
		return 1
	} else {
		return 0
	}
}

func flagToBool(b byte) bool {
	if b == 1 {
		return true
	} else {
		return false
	}
}
