package ppu

// object attribute memory/sprites
type OAMController struct {
	ByteIndexLatch uint16
	ByteIndex      uint16

	LowByteLatch byte

	LowTable  []byte
	HighTable []byte

	priorityRotation bool
}

func NewOAM() *OAMController {
	return &OAMController{
		LowTable:  make([]byte, 0x200),
		HighTable: make([]byte, 0x20)}
}

func (oam *OAMController) SetAddWordLow(value byte) {
	oam.ByteIndexLatch = (oam.ByteIndexLatch & 0x0100) | uint16(value)
	oam.ByteIndexLatch <<= 1
	oam.InvalidateInternalIndex()
}

func (oam *OAMController) SetAddWordHigh(value byte) {
	oam.ByteIndexLatch = (oam.ByteIndexLatch & 0xFF) | (uint16(value&1) << 8)
	oam.ByteIndexLatch <<= 1
	oam.InvalidateInternalIndex()
	oam.priorityRotation = value&0x80 == 0x80
}

func (oam *OAMController) WriteOAMData(value byte) {
	if !isOAMHighTable(oam.ByteIndex) {
		if oam.ByteIndex&1 == 1 {
			oam.LowTable[wrapOAMLowTableIndex(oam.ByteIndex)] = value
			oam.LowTable[wrapOAMLowTableIndex(oam.ByteIndex-1)] = oam.LowByteLatch
		} else {
			oam.LowByteLatch = value
		}
	} else {
		oam.HighTable[wrapOAMHighTableIndex(oam.ByteIndex)] = value
	}

	oam.ByteIndex++
}

func (oam *OAMController) ReadOAMData() byte {
	var ret byte

	if !isOAMHighTable(oam.ByteIndex) {
		ret = oam.LowTable[wrapOAMLowTableIndex(oam.ByteIndex)]
	} else {
		ret = oam.HighTable[wrapOAMHighTableIndex(oam.ByteIndex)]
	}

	oam.ByteIndex++
	return ret
}

func (oam *OAMController) InvalidateInternalIndex() {
	oam.ByteIndex = oam.ByteIndexLatch
}

func isOAMHighTable(index uint16) bool {
	return (index>>9)&1 == 1
}

func wrapOAMHighTableIndex(index uint16) uint16 {
	return index & 0x1F
}

func wrapOAMLowTableIndex(index uint16) uint16 {
	return index & 0x1FF
}
