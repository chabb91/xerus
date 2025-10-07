package ppu

type OAMController struct {
	OAMByteIndexLatch uint16
	OAMByteIndex      uint16

	OAMLowByteCache byte

	OAMLowTable  []byte
	OAMHighTable []byte

	priorityRotation bool
}

func (oam *OAMController) SetAddWordLow(value byte) {
	oam.OAMByteIndexLatch = (oam.OAMByteIndexLatch & 0x0100) | uint16(value)
	oam.OAMByteIndexLatch <<= 1
	oam.InvalidateInternalIndex()
}

func (oam *OAMController) SetAddWordHigh(value byte) {
	oam.OAMByteIndexLatch = (oam.OAMByteIndexLatch & 0xFF) | (uint16(value&1) << 8)
	oam.OAMByteIndexLatch <<= 1
	oam.InvalidateInternalIndex()
	oam.priorityRotation = value&0x80 == 1
}

func (oam *OAMController) WriteOAMData(value byte) {
	if !isOAMHighTable(oam.OAMByteIndex) {
		if oam.OAMByteIndex&1 == 1 {
			oam.OAMLowTable[wrapOAMLowTableIndex(oam.OAMByteIndex)] = value
			oam.OAMLowTable[wrapOAMLowTableIndex(oam.OAMByteIndex-1)] = oam.OAMLowByteCache
		} else {
			oam.OAMLowByteCache = value
		}
	} else {
		oam.OAMHighTable[wrapOAMHighTableIndex(oam.OAMByteIndex)] = value
	}

	oam.OAMByteIndex++
}

func (oam *OAMController) ReadOAMData() byte {
	var ret byte

	if !isOAMHighTable(oam.OAMByteIndex) {
		ret = oam.OAMLowTable[wrapOAMLowTableIndex(oam.OAMByteIndex)]
	} else {
		ret = oam.OAMHighTable[wrapOAMHighTableIndex(oam.OAMByteIndex)]
	}

	oam.OAMByteIndex++
	return ret
}

func (oam *OAMController) InvalidateInternalIndex() {
	oam.OAMByteIndex = oam.OAMByteIndexLatch
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
