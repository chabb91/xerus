package ppu

type Objects struct {
	ds tileDataSource

	Sprites [128]Sprite

	name, nameBase byte
	tileSize       [2]OBTileSize
}

func newObjects(ds tileDataSource) *Objects {
	ret := &Objects{ds: ds}
	for i := range 128 {
		ret.Sprites[i].ob = ret
		ret.Sprites[i].id = i
	}
	return ret
}

func (ob *Objects) setupOBSEL(value byte) {
	ob.tileSize = obTileSizeLUT[(value>>5)&0x7]
	ob.name = (value >> 3) & 0x3
	ob.nameBase = value & 0x7
}

func (ob *Objects) draw8sprites(H, V uint16) uint16 {
	for i := range byte(8) {
		if ret := ob.drawASprite(i, H, V); ret != 0 {
			return ret
		}
	}
	return 0
}

func (ob *Objects) drawASprite(value byte, H, V uint16) uint16 {
	sprite := &ob.Sprites[value&127]
	sprite.setup()
	dimensions := ob.tileSize[sprite.size]
	if sprite.posX <= int16(H) && uint16(sprite.posY) <= V &&
		sprite.posX+int16(dimensions.W) > int16(H) && uint16(sprite.posY)+dimensions.H > V {
		x := H - uint16(sprite.posX)
		y := V - uint16(sprite.posY)
		row := y >> 3
		column := x >> 3
		tileRow := ((sprite.tileIndex >> 4) + byte(row)) & 0xF
		tileColumn := (sprite.tileIndex + byte(column)) & 0xF
		tileIndex := tileRow<<4 | tileColumn
		wordIndex := sprite.GetVramTileWordIndex(tileIndex)

		var resolvedData [8][8]byte
		RenderTile4bppLUT(ob.ds.getVRAM(), uint16(wordIndex), &resolvedData)
		px := x & 7
		r := y & 7

		//fmt.Println(x, y)
		//fmt.Println(px, r)
		//fmt.Println(sprite)
		//fmt.Println(tileIndex)
		//fmt.Println(wordIndex)
		//fmt.Println(resolvedData[px][r])
		//return ob.ds.getCGRAM()[sprite.GetCgramIndex(int(resolvedData[r][px]))]
		return uint16(sprite.GetCgramIndex(int(resolvedData[r][px])))
	}
	//return ob.ds.getCGRAM()[0]
	return 0

}

type Sprite struct {
	id int
	ob *Objects

	posX int16
	posY byte

	tileIndex  byte
	nameTable  byte
	paletteNum byte
	priority   byte

	isFlippedHorizontally, isFlippedVertically bool
	size                                       byte
}

func (sprite *Sprite) setup() {
	id := sprite.id
	lowTable := sprite.ob.ds.getOAMLow()

	hi := (sprite.ob.ds.getOAMHigh()[id>>2] >> (byte(id&3) << 1)) & 0x03
	id *= 4
	lo3 := lowTable[id+3]

	sprite.posX = signExtend9(uint16(hi&1)<<8 | uint16(lowTable[id]))
	sprite.posY = lowTable[id+1]
	sprite.tileIndex = lowTable[id+2]
	sprite.nameTable = lo3 & 1
	sprite.paletteNum = (lo3 >> 1) & 0x7
	sprite.priority = (lo3 >> 4) & 0x3
	sprite.isFlippedVertically = (lo3>>7)&1 == 1
	sprite.isFlippedHorizontally = (lo3>>6)&1 == 1
	sprite.size = (hi >> 1) & 1
}

// converts the local palette index (0-15) to CGRAM index
func (sprite *Sprite) GetCgramIndex(localIndex int) int {
	localIndex %= 16
	return int(128 + sprite.paletteNum*16 + byte(localIndex))
}

// finds the first tile index belonging to this sprite in the VRAM
func (sprite *Sprite) GetVramFirstTileWordIndex() int {
	if sprite.nameTable == 0 {
		return int(((uint16(sprite.ob.nameBase) << 13) + (uint16(sprite.tileIndex) << 4)) & 0x7FFF)
	} else {
		return int(((uint16(sprite.ob.nameBase) << 13) + (uint16(sprite.tileIndex) << 4) + ((uint16(sprite.ob.name) + 1) << 12)) & 0x7FFF)
	}
}

func (sprite *Sprite) GetVramTileWordIndex(tileIndex byte) int {
	if sprite.nameTable == 0 {
		return int(((uint16(sprite.ob.nameBase) << 13) + (uint16(tileIndex) << 4)) & 0x7FFF)
	} else {
		return int(((uint16(sprite.ob.nameBase) << 13) + (uint16(tileIndex) << 4) + ((uint16(sprite.ob.name) + 1) << 12)) & 0x7FFF)
	}
}

func signExtend9(v uint16) int16 {
	v &= 0x1FF
	if v&0x100 != 0 {
		return int16(int32(v) - 0x200)
	}
	return int16(v)
}
