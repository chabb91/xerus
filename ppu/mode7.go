package ppu

type fillFunc func([]uint16) int

type Mode7 struct {
	ds tileDataSource

	m7A, m7B, m7C, m7D int16
	m7X, m7Y           int16 //13 bit twos complement signed
	hScroll, vScroll   uint16

	isFlippedHorizontally, isFlippedVertically bool
	fillFunc                                   fillFunc
}

func newMode7(ds tileDataSource) *Mode7 {
	return &Mode7{ds: ds}
}

func (bg *Mode7) GetDotAt(H, V uint16, _ bool) (int, byte, bool) {
	if bg.isFlippedHorizontally {
		V = 255 - V
	}
	if bg.isFlippedVertically {
		H = 255 - H
	}

	hScroll, vScroll := float32(int16(H+bg.hScroll))-float32(bg.m7X), float32(int16(V+bg.vScroll+(1<<interlace)))-float32(bg.m7Y)
	X := uint16(float32(bg.m7A)*hScroll/256.0 + float32(bg.m7B)*vScroll/256.0 + float32(bg.m7X))
	Y := uint16(float32(bg.m7C)*hScroll/256.0 + float32(bg.m7D)*vScroll/256.0 + float32(bg.m7Y))

	if bg.fillFunc == nil {
		X &= 1023
		Y &= 1023
	} else {
		if X > 1023 || Y > 1023 {
			return bg.fillFunc(bg.ds.getCGRAM()), 0, true
		}
	}

	vram := bg.ds.getVRAM()
	tile := vram[((Y&0xFFF8)<<4+(X>>3))] & 0x00FF
	char := byte(vram[((tile<<6)+((Y&7)<<3)+(X&7))] >> 8)
	var color int
	if char&255 == 0 {
		color = int(bg.ds.getCGRAM()[0])
	} else {
		color = int(bg.ds.getCGRAM()[char])
	}
	return color, 0, true
}

func (bg *Mode7) setM7Sel(value byte) {
	bg.isFlippedHorizontally = value&1 == 1
	bg.isFlippedVertically = value&2 == 1

	switch value & 0xC0 {
	case 0x80:
		bg.fillFunc = fillWithTransparent
	case 0xC0:
		bg.fillFunc = fillWithCharZero
	default:
		bg.fillFunc = nil
	}
}

func fillWithCharZero(cgram []uint16) int {
	return int(cgram[0])
}

// TODO rewrite this so it returns bg_transparent whatever
func fillWithTransparent(cgram []uint16) int {
	return int(cgram[0])
}

func signExtend13(v uint16) int16 {
	v &= 0x1FFF
	if v&0x1000 != 0 {
		return int16(int32(v) - 0x2000)
	}
	return int16(v)
}
