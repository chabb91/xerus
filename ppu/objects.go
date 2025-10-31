package ppu

const OBJ_CHAR_PER_TABLE = 256
const (
	BG_BACKDROP_COLOR  = 0
	OBJ_BACKDROP_COLOR = 128
)

type renderedSpriteOnDot struct {
	colorId             byte
	partakesInColorMath bool
}

type Objects struct {
	ds tileDataSource

	Sprites [128]Sprite

	spritesOnScanLine       [SCREEN_WIDTH]renderedSpriteOnDot
	participatingOnScanLine []*Sprite

	charTiles  [2][OBJ_CHAR_PER_TABLE]CharTile
	colorDepth colorDepth

	name, nameBase uint16
	tileSize       [2]OBTileSize

	currentEpoch *uint64

	layerId ppuLayer
}

func newObjects(ds tileDataSource, epochPtr *uint64, layer ppuLayer) *Objects {
	obj := &Objects{
		ds:                      ds,
		currentEpoch:            epochPtr,
		layerId:                 layer,
		colorDepth:              bpp4,
		participatingOnScanLine: make([]*Sprite, 32),
	}
	for i := range len(obj.Sprites) {
		obj.Sprites[i].ob = obj
		obj.Sprites[i].id = i
		obj.Sprites[i].isValid = false
	}

	for i := range 2 {
		for j := range OBJ_CHAR_PER_TABLE {
			obj.charTiles[i][j].layerEpoch = obj
			obj.charTiles[i][j].ds = obj.ds
			obj.charTiles[i][j].isValid = false
		}
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

// TODO work in progress. its not detecting the correct sprite prio, it doesnt count tiles rendered it counts X and Y wrong
// lots to do with this one
// TODO if OAM didnt change between frames all this can be cached. need a mechanism for that
func (ob *Objects) prepareScanLine(V uint16) {
	spriteCnt := 0
	tileCnt := int16(0)
	for i := range SCREEN_WIDTH {
		ob.spritesOnScanLine[i].colorId = OBJ_BACKDROP_COLOR
	}
	for i := range ob.Sprites {
		sprite := &ob.Sprites[i]
		sprite.setup()
		dimensions := ob.tileSize[sprite.size]
		//cast to byte so it wraps meaning high Y big sprites can wrap to the top
		if uint16(sprite.posY) <= V && uint16(sprite.posY+byte(dimensions.H)) > V {
			if (-1*int16(dimensions.W) < sprite.posX && sprite.posX < SCREEN_WIDTH) || sprite.posX == -256 {
				if spriteCnt == 32 {
					//TODO set $213E
					break
				}

				ob.participatingOnScanLine[spriteCnt] = sprite
				spriteCnt++
			}
		}
	}
	//this isnt 100% accurate because it can render more than 34 tiles but the flag is correctly set
	//and its going to behave roughly the same IF ITS NOT BUGGED
	visibleSprites := ob.participatingOnScanLine[:spriteCnt]
	spriteCnt = 0
	for i := len(visibleSprites) - 1; i >= 0; i-- {
		sprite := visibleSprites[i]
		dimensions := ob.tileSize[sprite.size]
		tileCnt += (min(SCREEN_WIDTH, sprite.posX+int16(dimensions.W)) - max(-8, sprite.posX)) >> 3
		spriteCnt++
		if tileCnt > 34 {
			//TODO set $213E
			break
		}
	}
	visibleSprites = visibleSprites[len(visibleSprites)-spriteCnt:]
	for _, sprite := range visibleSprites {
		dimensions := ob.tileSize[sprite.size]
		for j := range int16(dimensions.W) {
			screenPos := sprite.posX + j
			//TODO this should be replaced by the screen window check
			//prolly at the sprite count evaluation too
			if screenPos < int16(SCREEN_WIDTH) && screenPos >= 0 {
				if renderDot := &ob.spritesOnScanLine[screenPos]; renderDot.colorId == OBJ_BACKDROP_COLOR {
					renderDot.colorId = ob.drawASpriteByRef(sprite, dimensions, uint16(screenPos), V)
					renderDot.partakesInColorMath = sprite.paletteNum >= 4
				}
			}
		}
	}
}

func (ob *Objects) drawASpriteByRef(sprite *Sprite, dimensions OBTileSize, H, V uint16) byte {
	x := H - uint16(sprite.posX)
	//trying to handle wrapping of big sprites, UNTESTED
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

	px := tileFlipXLUT[sprite.flipIndex][x&7]
	r := tileFlipYLUT[sprite.flipIndex][y&7]

	char := &ob.charTiles[sprite.nameTable][tileIndex]

	return sprite.GetCgramIndex(char.getPixelAt(ob.colorDepth, sprite.GetVramTileWordIndex, tileIndex, px, r))
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
	lowTable := sprite.ob.ds.getOAMLow()

	hi := (sprite.ob.ds.getOAMHigh()[id>>2] >> (byte(id&3) << 1)) & 0x03
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
	return OBJ_BACKDROP_COLOR + sprite.paletteNum<<4 + localIndex
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
