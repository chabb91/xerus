package ppu

const OBJ_CHAR_PER_TABLE = 256

type renderedSpriteOnDot struct {
	color               int
	priority            byte
	partakesInColorMath bool
}

type Objects struct {
	priorityRotation func() byte

	CGRAM []uint16

	Sprites [128]Sprite

	resolvedDotsOnScanLine         [SCREEN_WIDTH]renderedSpriteOnDot
	spritesParticipatingOnScanLine []*Sprite

	charTiles  [2][OBJ_CHAR_PER_TABLE]CharTile
	colorDepth colorDepth

	name, nameBase uint16
	tileSize       [2]OBTileSize

	currentEpoch uint64

	layerId ppuLayer

	enabledOnMainScreen, enabledOnSubScreen bool

	//TODO fullsnes claims these flags are being set at a certain time gotta look into it
	//Bit6 when V=OBJ.YLOC/H=OAM.INDEX*2, bit7 when V=OBJ.YLOC+1/H=0
	timeOver, rangeOver byte
}

func newObjects(ds tileDataSource, priorityRotation func() byte, layer ppuLayer) *Objects {
	obj := &Objects{
		layerId:                        layer,
		colorDepth:                     bpp4,
		priorityRotation:               priorityRotation,
		CGRAM:                          ds.getCGRAM(),
		spritesParticipatingOnScanLine: make([]*Sprite, 32),
	}

	for i := range len(obj.Sprites) {
		obj.Sprites[i].id = i
		obj.Sprites[i].ob = obj
		obj.Sprites[i].OAMLow = ds.getOAMLow()
		obj.Sprites[i].OAMHigh = ds.getOAMHigh()
		obj.Sprites[i].isValid = false
	}

	for i := range 2 {
		for j := range OBJ_CHAR_PER_TABLE {
			obj.charTiles[i][j].layerEpoch = obj
			obj.charTiles[i][j].VRAM = ds.getVRAM()
			obj.charTiles[i][j].isValid = false
		}
	}

	return obj
}

func (ob *Objects) GetDotAt(H, _ uint16) (int, byte, bool) {
	ret := ob.resolvedDotsOnScanLine[H]
	return ret.color, ret.priority, ret.partakesInColorMath
}

func (ob *Objects) isActive() bool {
	return ob.enabledOnMainScreen || ob.enabledOnSubScreen
}

func (ob *Objects) resetTimeAndRange() {
	ob.timeOver, ob.rangeOver = 0, 0
}

func (ob *Objects) setupOBSEL(value byte) {
	ob.tileSize = obTileSizeLUT[(value>>5)&0x7]
	ob.name = (uint16((value>>3)&0x3) + 1) << 12
	ob.nameBase = uint16(value&0x7) << 13
}

func (ob *Objects) GetLayerSourceEpoch() uint64 {
	return ob.currentEpoch
}

func (ob *Objects) Invalidate(addr uint16) {
	baseAddress := ob.nameBase & 0x7FFF

	offsetAddress := (ob.nameBase + ob.name) & 0x7FFF

	if addr >= baseAddress && addr < baseAddress+0x1000 {
		tileIndex := (addr - baseAddress) >> 4
		if tileIndex < OBJ_CHAR_PER_TABLE {
			ob.charTiles[0][tileIndex].isValid = false
		}

	} else if addr >= offsetAddress && addr < offsetAddress+0x1000 {
		tileIndex := (addr - offsetAddress) >> 4
		if tileIndex < OBJ_CHAR_PER_TABLE {
			ob.charTiles[1][tileIndex].isValid = false
		}
	}
}

// TODO if OAM didnt change between frames all this can be cached. need a mechanism for that
// TODO implement the priority rotation oddity
func (ob *Objects) prepareScanLine(V uint16) {
	spriteCnt := 0
	tileCnt := int16(0)
	priority := ob.priorityRotation()
	for i := range ob.Sprites {
		sprite := &ob.Sprites[(priority+byte(i))&0x7F]
		sprite.setup()
		dimensions := ob.tileSize[sprite.size]
		wrapped := false
		posY := uint16(sprite.posY) + dimensions.H
		if posY > 255 {
			wrapped = true
		}
		posY &= 0xFF
		if (uint16(sprite.posY) <= V || wrapped) && posY > V {
			if (-1*int16(dimensions.W) < sprite.posX && sprite.posX < SCREEN_WIDTH) || sprite.posX == -256 {
				if spriteCnt == 32 {
					ob.rangeOver = 0x40
					break
				}

				ob.spritesParticipatingOnScanLine[spriteCnt] = sprite
				spriteCnt++
			}
		}
	}
	//this isnt 100% accurate because it can render more than 34 tiles but the flag is correctly set
	visibleSprites := ob.spritesParticipatingOnScanLine[:spriteCnt]
	spriteCnt = 0
	for i := len(visibleSprites) - 1; i >= 0; i-- {
		sprite := visibleSprites[i]
		dimensions := ob.tileSize[sprite.size]
		tileCnt += (min(SCREEN_WIDTH, sprite.posX+int16(dimensions.W)) - max(-8, sprite.posX)) >> 3
		spriteCnt++
		if tileCnt > 34 {
			ob.timeOver = 0x80
			break
		}
	}
	if !ob.isActive() {
		return
	}
	for i := range SCREEN_WIDTH {
		ob.resolvedDotsOnScanLine[i].color = BG_BACKDROP_COLOR
	}
	visibleSprites = visibleSprites[len(visibleSprites)-spriteCnt:]
	for _, sprite := range visibleSprites {
		dimensions := ob.tileSize[sprite.size]
		limit := sprite.posX + int16(dimensions.W)
		for screenPos := sprite.posX; screenPos < limit; {
			if screenPos < int16(SCREEN_WIDTH) && screenPos >= 0 {
				if ob.resolvedDotsOnScanLine[screenPos].color == BG_BACKDROP_COLOR {
					screenPos += ob.drawASpriteTileRow(sprite, dimensions, uint16(screenPos), V)
					continue
				}
			}
			screenPos++
		}
	}
}

func (ob *Objects) drawASpriteTileRow(sprite *Sprite, dimensions OBTileSize, H, V uint16) int16 {
	x := H - uint16(sprite.posX)
	y := uint16(sprite.posY)
	if V >= y {
		y = V - y
	} else {
		y = (256 - y) + V
	}

	row := y >> 3
	column := x >> 3
	//TODO doesnt flip accurately for the rectangular sprites but i couldnt care less at this moment
	if sprite.isFlippedHorizontally {
		column = dimensions.tilesPerRow - 1 - column
	}
	if sprite.isFlippedVertically {
		row = dimensions.tilesPerColumn - 1 - row
	}
	tileRow := ((sprite.tileIndex >> 4) + byte(row)) & 0xF
	tileColumn := (sprite.tileIndex + byte(column)) & 0xF
	tileIndex := tileRow<<4 | tileColumn

	px := x & 7
	flipXTtable := &tileFlipXLUT[sprite.flipIndex]
	r := tileFlipYLUT[sprite.flipIndex][y&7]

	cgram := ob.CGRAM

	char := &ob.charTiles[sprite.nameTable][tileIndex]
	rowData := char.getRowAt(ob.colorDepth, sprite.GetVramTileWordIndex, tileIndex, r)
	limit := min(8-px, SCREEN_WIDTH-H)

	for i := uint16(0); i < limit; i++ {
		renderDot := &ob.resolvedDotsOnScanLine[H+i]
		if renderDot.color != BG_BACKDROP_COLOR {
			continue
		}

		colorIndex := 128 + sprite.paletteNum<<4 + rowData[(flipXTtable[px+i])]
		if colorIndex&0xF == 0 {
			renderDot.color = BG_BACKDROP_COLOR
		} else {
			renderDot.color = int(cgram[colorIndex])
		}
		renderDot.priority = sprite.priority
		renderDot.partakesInColorMath = sprite.paletteNum >= 4
	}
	return int16(limit)
}

type Sprite struct {
	id int
	ob *Objects

	OAMHigh, OAMLow []byte

	posX int16
	posY byte

	tileIndex  byte
	nameTable  byte
	paletteNum byte
	priority   byte

	flipIndex                                  byte
	isFlippedHorizontally, isFlippedVertically bool
	size                                       byte

	isValid bool
}

func (sprite *Sprite) setup() {
	if sprite.isValid {
		return
	}

	//apparently this thing is GIGA heavy
	id := sprite.id
	lowTable := sprite.OAMLow

	hi := (sprite.OAMHigh[id>>2] >> (byte(id&3) << 1)) & 0x03
	id <<= 2
	lo3 := lowTable[id+3]

	sprite.posX = signExtend9(uint16(hi&1)<<8 | uint16(lowTable[id]))
	sprite.posY = lowTable[id+1]
	sprite.tileIndex = lowTable[id+2]
	sprite.nameTable = lo3 & 1
	sprite.paletteNum = (lo3 >> 1) & 0x7
	sprite.priority = (lo3 >> 4) & 0x3
	sprite.isFlippedVertically = (lo3>>7)&1 == 1
	sprite.isFlippedHorizontally = (lo3>>6)&1 == 1
	sprite.flipIndex = (lo3 >> 6) & 3
	sprite.size = (hi >> 1) & 1

	sprite.isValid = true
}

// converts the local palette index (0-15) to CGRAM index
func (sprite *Sprite) GetCgramIndex(localIndex byte) byte {
	localIndex &= 15
	return 128 + sprite.paletteNum<<4 + localIndex
}

// finds the first tile index belonging to this sprite in the VRAM
func (sprite *Sprite) GetVramFirstTileWordIndex() int {
	if sprite.nameTable == 0 {
		return int((sprite.ob.nameBase + (uint16(sprite.tileIndex) << 4)) & 0x7FFF)
	} else {
		return int((sprite.ob.nameBase + (uint16(sprite.tileIndex) << 4) + sprite.ob.name) & 0x7FFF)
	}
}

func (sprite *Sprite) GetVramTileWordIndex(tileIndex byte) uint16 {
	if sprite.nameTable == 0 {
		return (sprite.ob.nameBase + (uint16(tileIndex) << 4)) & 0x7FFF
	} else {
		return (sprite.ob.nameBase + (uint16(tileIndex) << 4) + sprite.ob.name) & 0x7FFF
	}
}

func signExtend9(v uint16) int16 {
	v &= 0x1FF
	if v&0x100 != 0 {
		return int16(int32(v) - 0x200)
	}
	return int16(v)
}
