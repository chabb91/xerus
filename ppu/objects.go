package ppu

type Objects struct {
	ds tileDataSource

	Sprites    [128]Sprite
	charTiles  [2][256]CharTile
	colorDepth colorDepth

	name, nameBase uint16
	tileSize       [2]OBTileSize

	currentEpoch *uint64

	layerId ppuLayer
}

func newObjects(ds tileDataSource, epochPtr *uint64, layer ppuLayer) *Objects {
	obj := &Objects{
		ds:           ds,
		currentEpoch: epochPtr,
		layerId:      layer,
		colorDepth:   bpp4,
	}
	for i := range 128 {
		obj.Sprites[i].ob = obj
		obj.Sprites[i].id = i
	}

	return obj
}

func (ob *Objects) setupOBSEL(value byte) {
	ob.tileSize = obTileSizeLUT[(value>>5)&0x7]
	ob.name = (uint16((value>>3)&0x3) + 1) << 12
	ob.nameBase = uint16(value&0x7) << 13
}

func (ob *Objects) GetLayerSourceEpoch() *uint64 {
	return ob.currentEpoch
}

func (ob *Objects) Invalidate(addr uint16) {
	baseAddress := ob.nameBase & 0x7FFF

	offsetAddress := (ob.nameBase + ob.name) & 0x7FFF

	if addr >= baseAddress && addr < baseAddress+0x1000 {
		tileIndex := (addr - baseAddress) >> 4
		if tileIndex < 256 {
			ob.charTiles[0][tileIndex].isValid = false
		}

	} else if addr >= offsetAddress && addr < offsetAddress+0x1000 {
		tileIndex := (addr - offsetAddress) >> 4
		if tileIndex < 256 {
			ob.charTiles[1][tileIndex].isValid = false
		}
	}
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

	isValid bool
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
	localIndex &= 15
	return int(128 + sprite.paletteNum<<4 + byte(localIndex))
}

// finds the first tile index belonging to this sprite in the VRAM
func (sprite *Sprite) GetVramFirstTileWordIndex() int {
	if sprite.nameTable == 0 {
		return int((sprite.ob.nameBase + (uint16(sprite.tileIndex) << 4)) & 0x7FFF)
	} else {
		return int((sprite.ob.nameBase + (uint16(sprite.tileIndex) << 4) + sprite.ob.name) & 0x7FFF)
	}
}

func (sprite *Sprite) GetVramTileWordIndex(tileIndex byte) int {
	if sprite.nameTable == 0 {
		return int((sprite.ob.nameBase + (uint16(tileIndex) << 4)) & 0x7FFF)
	} else {
		return int((sprite.ob.nameBase + (uint16(tileIndex) << 4) + sprite.ob.name) & 0x7FFF)
	}
}

func signExtend9(v uint16) int16 {
	v &= 0x1FF
	if v&0x100 != 0 {
		return int16(int32(v) - 0x200)
	}
	return int16(v)
}
