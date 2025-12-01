package ppu

// Color/Palette RAM
type CGRAMController struct {
	WordIndex   byte
	byteCounter byte

	LowByteLatch byte

	CGRAM []uint16

	ppu2OB *byte
}

func NewCGRAM(ppu2OB *byte) *CGRAMController {
	return &CGRAMController{
		CGRAM:  make([]uint16, 0x100),
		ppu2OB: ppu2OB,
	}
}

func (cgram *CGRAMController) SetAddWord(value byte) {
	cgram.WordIndex = value
	cgram.byteCounter = 0
}

func (cgram *CGRAMController) WriteData(value byte) {
	if cgram.byteCounter&1 == 1 {
		cgram.CGRAM[cgram.WordIndex] = uint16(value)<<8 | uint16(cgram.LowByteLatch)
		cgram.WordIndex++
	} else {
		cgram.LowByteLatch = value
	}
	cgram.byteCounter++
}

func (cgram *CGRAMController) ReadData() byte {
	var ret byte

	if cgram.byteCounter&1 == 1 {
		ret = *cgram.ppu2OB&0x80 | byte(cgram.CGRAM[cgram.WordIndex]>>8)&0x7F
		cgram.WordIndex++
	} else {
		ret = byte(cgram.CGRAM[cgram.WordIndex])
	}
	cgram.byteCounter++
	return ret
}
