package ppu

const (
	M7_REPEAT = iota
	M7_FILL_WITH_CHAR_ZERO
	M7_FILL_WITH_TRANSPARENT
)

type Mode7 struct {
	ds tileDataSource

	m7A, m7B, m7C, m7D int16
	m7X, m7Y           int16 //13 bit twos complement signed
	hScroll, vScroll   int16 //13 bit twos complement signed

	isFlippedHorizontally, isFlippedVertically bool
	fillMode                                   int

	bg1Mosaic, bg2Mosaic, isDirectColor *bool

	characterDataOnScanLine [SCREEN_WIDTH]byte
}

func newMode7(ds tileDataSource, bg1, bg2 *Background) *Mode7 {
	return &Mode7{
		ds:            ds,
		isDirectColor: &bg1.isDirectColor,
		bg1Mosaic:     &bg1.mosaic,
		bg2Mosaic:     &bg2.mosaic,
	}
}

func (bg *Mode7) prepareScanLine(V uint16) {
	if *bg.bg1Mosaic {
		V = V - V%uint16(mosaicSize)
	}
	hFlipMask := byte(0)
	if bg.isFlippedHorizontally {
		hFlipMask = 0xFF
	}
	if bg.isFlippedVertically {
		//is this how it should be who knows
		V = (256 << interlace) - 1 - V
	}
	vram := bg.ds.getVRAM()

	dx := clip(int32(bg.hScroll) - int32(bg.m7X))
	dy := clip(int32(bg.vScroll) - int32(bg.m7Y))

	X0 := ((int32(bg.m7A) * dx) &^ 63) + ((int32(bg.m7B) * int32(V)) &^ 63) +
		((int32(bg.m7B) * dy) &^ 63) + (int32(bg.m7X) << 8)
	Y0 := ((int32(bg.m7C) * dx) &^ 63) + ((int32(bg.m7D) * int32(V)) &^ 63) +
		((int32(bg.m7D) * dy) &^ 63) + (int32(bg.m7Y) << 8)

	for i := range SCREEN_WIDTH {
		X, Y := uint16(X0>>8), uint16(Y0>>8)

		//increment after so X[0], Y[0] are preserved
		X0 += int32(bg.m7A)
		Y0 += int32(bg.m7C)

		idx := byte(i) ^ hFlipMask // (xor with FF is the same as 255-i)
		tile := uint16(0)
		if bg.fillMode == M7_REPEAT || (X <= 1023 && Y <= 1023) {
			X &= 1023
			Y &= 1023
			tile = vram[((Y&0xFFF8)<<4|(X>>3))] & 0xFF
		} else if bg.fillMode == M7_FILL_WITH_TRANSPARENT {
			bg.characterDataOnScanLine[idx] = 0
			continue
			//else fill with char0 i. e. tile is 0
		}

		bg.characterDataOnScanLine[idx] = byte(vram[((tile<<6)|((Y&7)<<3)|(X&7))] >> 8)
	}
}

func (bg *Mode7) GetDotAt(H, _ uint16) (int, byte, bool) {
	if *bg.bg1Mosaic {
		H = H - H%uint16(mosaicSize)
	}

	char := bg.characterDataOnScanLine[H]

	var color int
	if char == 0 {
		color = BG_BACKDROP_COLOR
	} else {
		if *bg.isDirectColor {
			red := char & 0x07
			green := char & 0x38
			blue := char & 0xC0
			color = int(uint16(blue)<<7 | uint16(green)<<4 | uint16(red)<<2)
		} else {
			color = int(bg.ds.getCGRAM()[char])
		}
	}
	return color, 1, true
}

func (bg *Mode7) GetDotAtEXTBG(H, _ uint16) (int, byte, bool) {
	if *bg.bg2Mosaic {
		H = H - H%uint16(mosaicSize)
	}

	char := bg.characterDataOnScanLine[H]

	if colorId := char & 0x7F; colorId == 0 {
		return BG_BACKDROP_COLOR, 0, true
	} else {
		color := bg.ds.getCGRAM()[colorId]
		return int(color), (char & 0x80 >> 7), true
	}
}

func (bg *Mode7) setM7Sel(value byte) {
	bg.isFlippedHorizontally = value&1 == 1
	bg.isFlippedVertically = value&2 == 2

	switch value & 0xC0 {
	case 0x80:
		bg.fillMode = M7_FILL_WITH_TRANSPARENT
	case 0xC0:
		bg.fillMode = M7_FILL_WITH_CHAR_ZERO
	default:
		bg.fillMode = M7_REPEAT
	}
}

func signExtend13(v uint16) int16 {
	v &= 0x1FFF
	if v&0x1000 != 0 {
		return int16(int32(v) - 0x2000)
	}
	return int16(v)
}

func clip(a int32) int32 {
	if a&0x2000 != 0 {
		return a | ^0x3FF
	}
	return a & 0x3FF
}
