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
func rel16(cpu *CPU, high, low byte) {
	cpu.r.PC += uint16(int16(createWord(high, low)))
}

// Most likely the proper addressing logic for the Direct Page mode.
// One main functions and 3 wrappers for convenience
func directPageLogic(cpu *CPU, op byte, register uint16, isPEI bool) (addressLo, addressHi, addressBank uint32) {
	if cpu.isW() && cpu.r.E && !isPEI {
		//according to my test data even indirect_long is affected by this
		//however the documentation doesnt mention it so
		//TODO keep an eye out for this not working right with indirect
		low := getLowByte(uint16(op) + register)
		addressLo = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low))
		addressHi = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low+1))
		addressBank = mapOffsetToBank(0x00, createWord(getHighByte(cpu.r.D), low+2))
	} else {
		offset := cpu.r.D + uint16(op) + register
		addressLo = mapOffsetToBank(0x00, offset)
		addressHi = mapOffsetToBank(0x00, offset+1)
		addressBank = mapOffsetToBank(0x00, offset+2)
	}

	return addressLo, addressHi, addressBank
}

func directPage(cpu *CPU, op byte, isPEI bool) (addressLo, addressHi uint32) {
	addressLo, addressHi, _ = directPageLogic(cpu, op, 0, isPEI)
	return addressLo, addressHi
}

func directPageXY(cpu *CPU, op byte, register uint16) (addressLo, addressHi uint32) {
	addressLo, addressHi, _ = directPageLogic(cpu, op, register, false)
	return addressLo, addressHi
}

func directPageLong(cpu *CPU, op byte) (uint32, uint32, uint32) {
	return directPageLogic(cpu, op, 0, false)
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

// Absolute long: [ba:hi:lo]
func absl(cpu *CPU, lo, hi, ba byte) uint32 { return createAddress(lo, hi, ba) }

// Absolute long,X: 24-bit wrap
func abslx(cpu *CPU, lo, hi, ba byte, X uint16) uint32 {
	return mask24(createAddress(lo, hi, ba) + uint32(X))
}

// (dp)          → pointer = [00:(D+dp) & 0xFFFF] (lo at dp, hi at dp+1 with 8-bit dp wrap), bank=DBR
// You must fetch p0 = ptr.lo at D+dp, p1 = ptr.hi at D+(dp+1)&0xFF (same high-byte of D)
func ind_dp(cpu *CPU, op1 byte, p0, p1 byte) uint32 {
	return mapOffsetToBank(cpu.r.DB, createWord(p0, p1))
}

// (dp,X)        → pointer addr base = D + ((dp + X) & 0xFF) (DP wrap on low byte), then read lo/hi there
func ind_dpx(cpu *CPU, op1 byte, X uint16, p0, p1 byte) uint32 {
	// you already did DP wrap while reading p0,p1 in your microcode
	return mapOffsetToBank(cpu.r.DB, createWord(p0, p1))
}

// (dp),Y        → pointer = [00:D+dp] (DP wrap for p0/p1), then add Y (16-bit wrap), bank=DBR
func ind_dp_y(cpu *CPU, op1 byte, Y uint16, p0, p1 byte) uint32 {
	return mapOffsetToBank(cpu.r.DB, createWord(p0, p1)+Y)
}

// [dp]          → long pointer = lo,hi,bank read at D+dp, D+dp+1, D+dp+2 (each 8-bit DP wrap)
// target = [ba:hi:lo]
func lind_dp(cpu *CPU, op1 byte, p0, p1, p2 byte) uint32 { // p2=bank
	return createAddress(p0, p1, p2)
}

// [dp],Y        → long pointer + Y (24-bit wrap)
func lind_dp_y(cpu *CPU, op1 byte, Y uint16, p0, p1, p2 byte) uint32 {
	return mask24(createAddress(p0, p1, p2) + uint32(Y))
}

// (abs)  16-bit pointer stored at [PBR:abs], target in same bank as pointer for JMP (abs)
// For data ops you rarely use this; for JMP (abs): bank = PBR (no bank change).
func ind_abs_sameBank(cpu *CPU, lo, hi byte, p0, p1 byte) uint32 {
	// You fetched p0 (lo), p1 (hi) from PBR:abs and PBR:abs+1 (16-bit wrap of low part).
	return mapOffsetToBank(cpu.r.PB, createWord(p0, p1))
}

// [abs]  24-bit long pointer at [PBR:abs..abs+2]
func lind_abs(cpu *CPU, lo, hi byte, p0, p1, p2 byte) uint32 {
	return createAddress(p0, p1, p2)
}

// dp,S       → effective = bank 0x00, offset = (S.low + op1) with 8-bit wrap, high byte from S page (native) or 0x01 (emulation).
// In practice, most emulators form a 16-bit address: (S_base & 0xFF00) | ((S.low + op1) & 0xFF)
func sr(cpu *CPU, op1 byte) uint32 {
	sLow := byte(cpu.r.S & 0x00FF)
	off8 := byte(uint16(sLow) + uint16(op1)) // 8-bit wrap
	var pageHi byte
	if cpu.r.E {
		pageHi = 0x01
	} else {
		pageHi = byte(cpu.r.S >> 8)
	}
	offs := createWord(off8, pageHi)
	return mapOffsetToBank(0x00, offs)
}

// (dp,S),Y   → first form SR (above) to read 16-bit pointer lo/hi at that address (and +1), then add Y; bank = DBR
func ind_sr_y(cpu *CPU, op1 byte, Y uint16, p0, p1 byte) uint32 {
	return mapOffsetToBank(cpu.r.DB, createWord(p0, p1)+Y)
}
