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

func (bgofs *BGxnOFS) setBg1V(bg1 *Background1, value byte) {
	bg1.vScroll = bgofs.vFormula(value)
}

func (bgofs *BGxnOFS) setBg1H(bg1 *Background1, value byte) {
	bg1.hScroll = bgofs.hFormula(value)
}
