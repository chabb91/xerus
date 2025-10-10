package ppu

// object attribute memory/sprites
type OAMController struct {
	obsel *OBSEL

	ByteIndexLatch uint16
	ByteIndex      uint16

	LowByteLatch byte

	LowTable  []byte
	HighTable []byte

	priorityRotation bool
}

func NewOAM() *OAMController {
	return &OAMController{
		obsel:     &OBSEL{},
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
	oam.priorityRotation = value&0x80 == 1
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

type OBSize struct {
	w, h byte
}

func (obsize *OBSize) Set(width, height byte) {
	obsize.w = width
	obsize.h = height
}

type OBSEL struct {
	smallForm, largeForm OBSize

	name, nameBase byte
}

func (obsel *OBSEL) Setup(value byte) {
	switch (value >> 5) & 0x7 {
	case 0:
		obsel.smallForm.Set(8, 8)
		obsel.largeForm.Set(16, 16)
	case 1:
		obsel.smallForm.Set(8, 8)
		obsel.largeForm.Set(32, 32)
	case 2:
		obsel.smallForm.Set(8, 8)
		obsel.largeForm.Set(64, 64)
	case 3:
		obsel.smallForm.Set(16, 16)
		obsel.largeForm.Set(32, 32)
	case 4:
		obsel.smallForm.Set(16, 16)
		obsel.largeForm.Set(64, 64)
	case 5:
		obsel.smallForm.Set(32, 32)
		obsel.largeForm.Set(64, 64)
	case 6:
		obsel.smallForm.Set(16, 32)
		obsel.largeForm.Set(32, 64)
	case 7:
		obsel.smallForm.Set(16, 32)
		obsel.largeForm.Set(32, 32)
	}
	obsel.name = (value >> 3) & 0x3
	obsel.nameBase = value & 0x7
}

type Sprite struct {
	id    int
	obsel *OBSEL

	posX int16
	posY byte

	tileIndex  byte
	nameTable  byte
	paletteNum byte
	priority   byte

	isFlippedHorizontally, isFlippedVertically bool
	isLarge                                    bool
}

func (oam *OAMController) NewSprite(recordId int) *Sprite {
	recordId %= 128
	ret := &Sprite{id: recordId}

	hi := (oam.HighTable[recordId/4] >> (byte(recordId%4) * 2)) & 0x03

	recordId *= 4
	lo3 := oam.LowTable[recordId+3]

	ret.obsel = oam.obsel
	ret.posX = signExtend9(uint16(hi&1)<<8 | uint16(oam.LowTable[recordId]))
	ret.posY = oam.LowTable[recordId+1]
	ret.tileIndex = oam.LowTable[recordId+2]
	ret.nameTable = lo3 & 1
	ret.paletteNum = (lo3 >> 1) & 0x7
	ret.priority = (lo3 >> 4) & 0x3
	ret.isFlippedVertically = (lo3>>7)&1 == 1
	ret.isFlippedHorizontally = (lo3>>6)&1 == 1
	ret.isLarge = (hi>>1)&1 == 1

	return ret
}

// converts the local palette index (0-15) to CGRAM index
func (sprite Sprite) GetCgramIndex(localIndex int) int {
	localIndex %= 16
	return int(128 + sprite.paletteNum*16 + byte(localIndex))
}

// finds the first tile index belonging to this sprite in the VRAM
func (sprite Sprite) GetVramFirstTileWordIndex() int {
	if sprite.nameTable == 0 {
		return int(((uint16(sprite.obsel.nameBase) << 13) + (uint16(sprite.tileIndex) << 4)) & 0x7FFF)
	} else {
		return int(((uint16(sprite.obsel.nameBase) << 13) + (uint16(sprite.tileIndex) << 4) + ((uint16(sprite.obsel.name) + 1) << 12)) & 0x7FFF)
	}
}

func signExtend9(v uint16) int16 {
	v &= 0x1FF
	if v&0x100 != 0 {
		return int16(int32(v) - 0x200)
	}
	return int16(v)
}
