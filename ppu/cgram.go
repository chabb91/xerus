package ppu

type CGRAMController struct {
	WordIndex   byte
	byteCounter byte

	LowByteLatch byte

	CGRAM []uint16
}

func NewCGRAM() *CGRAMController {
	return &CGRAMController{
		CGRAM: make([]uint16, 0x100),
	}
}

func (cgram *CGRAMController) SetAddWord(value byte) {
	cgram.WordIndex = value
	cgram.byteCounter = 0
}

func (cgram *CGRAMController) WriteOAMData(value byte) {
	if cgram.byteCounter&1 == 1 {
		cgram.CGRAM[cgram.WordIndex] = uint16(value)<<8 | uint16(cgram.LowByteLatch)
		cgram.WordIndex++
	} else {
		cgram.LowByteLatch = value
	}
	cgram.byteCounter++
}

func (cgram *CGRAMController) ReadOAMData() byte {
	var ret byte

	if cgram.byteCounter&1 == 1 {
		ret = byte(cgram.CGRAM[cgram.WordIndex] >> 8)
		cgram.WordIndex++
	} else {
		ret = byte(cgram.CGRAM[cgram.WordIndex])
	}
	cgram.byteCounter++
	return ret
}
