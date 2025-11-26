package ppu

type BGxnOFS struct {
	prev1 byte //shared by all
	prev2 byte // shared by all horizontal registers
}

func (bgofs *BGxnOFS) hFormula(current byte) uint16 {
	ret := (uint16(current) << 8) | (uint16(bgofs.prev1 & 0xF8)) | uint16(bgofs.prev2&7)
	bgofs.prev1, bgofs.prev2 = current, current

	return ret & 0x03FF
}

func (bgofs *BGxnOFS) vFormula(current byte) uint16 {
	ret := (uint16(current) << 8) | uint16(bgofs.prev1)
	bgofs.prev1 = current

	return ret & 0x03FF
}

type M7Registers struct {
	prev byte //shared by all m7 registers
}

func (m7 *M7Registers) setRegister(current byte) uint16 {
	register := uint16(current)<<8 | uint16(m7.prev)
	m7.prev = current

	return register
}
