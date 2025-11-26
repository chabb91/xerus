package ppu

type Mode7 struct {
	ds tileDataSource

	m7A, m7B, m7C, m7D int16
	m7X, m7Y           int16 //13 bit twos complement signed
	hScroll, vScroll   uint16
}

func newMode7(ds tileDataSource) *Mode7 {
	return &Mode7{ds: ds}
}

func (bg *Mode7) GetDotAt(H, V uint16, _ bool) (int, byte, bool) {
	hScroll, vScroll := int16(H+bg.hScroll)-bg.m7X, int16(V+bg.vScroll+(1<<interlace))-bg.m7Y
	X := uint16(float32(bg.m7A*hScroll)/256.0 + float32(bg.m7B*vScroll)/256.0 + float32(bg.m7X))
	Y := uint16(float32(bg.m7C*hScroll)/256.0 + float32(bg.m7D*vScroll)/256.0 + float32(bg.m7Y))

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

func signExtend13(v uint16) int16 {
	v &= 0x1FFF
	if v&0x1000 != 0 {
		return int16(int32(v) - 0x2000)
	}
	return int16(v)
}
