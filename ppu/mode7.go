package ppu

type fillFunc func([]uint16) int

type Mode7 struct {
	ds tileDataSource

	m7A, m7B, m7C, m7D int16
	m7X, m7Y           int16 //13 bit twos complement signed
	hScroll, vScroll   uint16

	isFlippedHorizontally, isFlippedVertically bool
	fillFunc                                   fillFunc

	bg1Mosaic, bg2Mosaic, isDirectColor *bool
}

func newMode7(ds tileDataSource, bg1, bg2 *Background) *Mode7 {
	return &Mode7{
		ds:            ds,
		isDirectColor: &bg1.isDirectColor,
		bg1Mosaic:     &bg1.mosaic,
		bg2Mosaic:     &bg2.mosaic,
	}
}

func (bg *Mode7) GetDotAt(H, V uint16) (int, byte, bool) {
	if *bg.bg1Mosaic {
		V = V - V%uint16(mosaicSize)
		H = H - H%uint16(mosaicSize)
	}

	char, outsideCanvas := bg.getCharTile(H, V)
	if outsideCanvas {
		return bg.fillFunc(bg.ds.getCGRAM()), 1, true
	}

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

func (bg *Mode7) GetDotAtEXTBG(H, V uint16) (int, byte, bool) {
	if *bg.bg1Mosaic {
		V = V - V%uint16(mosaicSize)
	}
	if *bg.bg2Mosaic {
		H = H - H%uint16(mosaicSize)
	}

	char, outsideCanvas := bg.getCharTile(H, V)
	if outsideCanvas {
		return bg.fillFunc(bg.ds.getCGRAM()), 0, true
	}

	if char&127 == 0 {
		return BG_BACKDROP_COLOR, 0, true
	} else {
		color := bg.ds.getCGRAM()[char&0x7F]
		return int(color), (char & 0x80 >> 7), true
	}
}

func (bg *Mode7) getCharTile(H, V uint16) (byte, bool) {
	//which is horizontal which is vertical who knows
	if bg.isFlippedHorizontally {
		H = 255 - H
	}
	if bg.isFlippedVertically {
		//is this how it should be who knows
		V = (256 << interlace) - 1 - V
	}

	hScroll, vScroll := float32(int16(H+bg.hScroll)-bg.m7X), float32(int16(V+bg.vScroll+(1<<interlace))-bg.m7Y)
	X := uint16((float32(bg.m7A)*hScroll+float32(bg.m7B)*vScroll)/256.0 + float32(bg.m7X))
	Y := uint16((float32(bg.m7C)*hScroll+float32(bg.m7D)*vScroll)/256.0 + float32(bg.m7Y))

	if bg.fillFunc == nil {
		X &= 1023
		Y &= 1023
	} else {
		if X > 1023 || Y > 1023 {
			return 0xFF, true
		}
	}

	vram := bg.ds.getVRAM()
	tile := vram[((Y&0xFFF8)<<4+(X>>3))] & 0x00FF
	return byte(vram[((tile<<6)+((Y&7)<<3)+(X&7))] >> 8), false
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

func fillWithTransparent(_ []uint16) int {
	return BG_BACKDROP_COLOR
}

func signExtend13(v uint16) int16 {
	v &= 0x1FFF
	if v&0x1000 != 0 {
		return int16(int32(v) - 0x2000)
	}
	return int16(v)
}
